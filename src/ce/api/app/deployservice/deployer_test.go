package deployservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/tasks"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type DeploySuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *DeploySuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// clean up left-over queue
	insp := tasks.Inspector()
	insp.DeleteQueue(tasks.QueueDeployService, true)
}

func (s *DeploySuite) AfterTest(_, _ string) {
	s.conn.CloseTx()

	// clean up newly created queue
	insp := tasks.Inspector()
	insp.DeleteQueue(tasks.QueueDeployService, true)
}

func (s *DeploySuite) Test_Deployment() {
	config.Get().Secrets["TEST_KEY"] = "test-value"

	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	depl := s.MockDeployment(env, map[string]any{
		"MigrationsPath": null.StringFrom("/migrations"),
		"BuildConfig": &buildconf.BuildConf{
			BuildCmd: "npm run build",
			StatusChecks: []buildconf.StatusCheck{
				{Name: "E2E", Cmd: "npm run test:e2e", Description: "Run e2e tests"},
			},
			Vars: map[string]string{
				"KEY_1":    "VAL_1",
				"KEY_2":    "$KEY_1",
				"KEY_3":    "$KEY_NONE",
				"TEST_KEY": "$TEST_KEY",
			},
		},
	})

	err := deployservice.New().Deploy(context.Background(), app.App, depl.Deployment)

	s.NoError(err)

	insp := tasks.Inspector()
	info, err := insp.GetTaskInfo(tasks.QueueDeployService, fmt.Sprintf("deployment-%s", depl.ID.String()))

	s.NoError(err)

	message, err := deployservice.FromEncrypted(string(info.Payload))

	s.NoError(err)

	s.Equal(&deployservice.DeploymentMessage{
		Client: deployservice.ClientConfig{
			Repo:        "https://github.com/svedova/react-minimal.git",
			Slug:        "svedova/react-minimal",
			AccessToken: "some-token",
		},
		Build: deployservice.BuildConfig{
			Env:            "production",
			Branch:         "main",
			BuildCmd:       "npm run build",
			APIFolder:      "/api",
			ShouldPublish:  false,
			EnvID:          env.ID.String(),
			AppID:          app.ID.String(),
			DeploymentID:   depl.ID.String(),
			MigrationsPath: "/migrations",
			StatusChecks: []buildconf.StatusCheck{
				{Name: "E2E", Cmd: "npm run test:e2e", Description: "Run e2e tests"},
			},
			Vars: map[string]string{
				"KEY_1":             "VAL_1",
				"KEY_2":             "VAL_1",
				"KEY_3":             "$KEY_NONE",
				"TEST_KEY":          "test-value",
				"SK_APP_ID":         depl.AppID.String(),
				"SK_DEPLOYMENT_ID":  depl.ID.String(),
				"SK_DEPLOYMENT_URL": fmt.Sprintf("http://%s--%s.stormkit:8888", app.DisplayName, depl.ID.String()),
				"SK_ENV":            "production",
				"SK_ENV_ID":         depl.EnvID.String(),
				"SK_ENV_URL":        fmt.Sprintf("http://%s.stormkit:8888", app.DisplayName),
				"STORMKIT":          "true",
			},
		},
		Config: &config.RunnerConfig{
			Provider:      "filesys",
			Concurrency:   0,
			MaxGoRoutines: 25,
		},
		Canary: nil,
	}, message)

	// Should also insert into database
	d, err := deploy.NewStore().DeploymentByID(context.Background(), depl.ID)
	s.NoError(err)
	s.Equal(depl.ID, d.ID)
	s.Equal(depl.Branch, d.Branch)
}

func (s *DeploySuite) Test_Deployment_NoMoreBuildMinutes() {
	config.SetIsStormkitCloud(true)
	usr := s.MockUser()
	app := s.MockApp(usr)

	_, err := s.conn.Exec(`
		INSERT INTO user_metrics (
			user_id,
			build_minutes,
			month,
			year
		) VALUES (
			$1,
			1250,
			EXTRACT(MONTH FROM NOW() AT TIME ZONE 'UTC'),
			EXTRACT(YEAR FROM NOW() AT TIME ZONE 'UTC')
		);
	`, usr.ID)

	s.NoError(err)

	deployer := &deployservice.DefaultDeployer{}
	err = deployer.Deploy(context.Background(), app.App, &deploy.Deployment{
		AppID: app.ID,
	})

	s.Error(err)
	s.Equal(deployservice.ErrBuildMinutesExceeded, err)
}

func TestAppDeploy(t *testing.T) {
	suite.Run(t, &DeploySuite{})
}
