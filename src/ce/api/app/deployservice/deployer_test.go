package deployservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
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
		"MigrationsFolder": null.StringFrom("/migrations"),
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
			Env:              "production",
			Branch:           "main",
			BuildCmd:         "npm run build",
			APIFolder:        "/api",
			ShouldPublish:    false,
			EnvID:            env.ID.String(),
			AppID:            app.ID.String(),
			DeploymentID:     depl.ID.String(),
			MigrationsFolder: "/migrations",
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
		Config: func() *deployservice.RunnerSettings {
			// Concurrency is `json:"-"` so it doesn't round-trip through the
			// encrypted payload — zero it out on the expected value.
			rc := *config.Get().Runner
			rc.Concurrency = 0
			return &deployservice.RunnerSettings{RunnerConfig: rc, AutoInstall: true}
		}(),
	}, message)

	// Should also insert into database
	d, err := deploy.NewStore().DeploymentByID(context.Background(), depl.ID)
	s.NoError(err)
	s.Equal(depl.ID, d.ID)
	s.Equal(depl.Branch, d.Branch)
}

// enqueueCapture is a TaskClient that records the enqueued task instead of
// sending it to Redis, so tests can inspect the deployment message without
// relying on shared queue state.
type enqueueCapture struct {
	task *asynq.Task
}

func (c *enqueueCapture) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	c.task = task
	return &asynq.TaskInfo{}, nil
}

func (s *DeploySuite) deployAndReadMessage(a *app.App, d *deploy.Deployment) *deployservice.DeploymentMessage {
	capture := &enqueueCapture{}
	prevClient := tasks.Client
	tasks.Client = func() tasks.TaskClient { return capture }

	defer func() { tasks.Client = prevClient }()

	deployer := &deployservice.DefaultDeployer{}
	s.Require().NoError(deployer.Deploy(context.Background(), a, d))
	s.Require().NotNil(capture.task)

	message, err := deployservice.FromEncrypted(string(capture.task.Payload()))
	s.Require().NoError(err)

	return message
}

func (s *DeploySuite) Test_Deployment_BuildCache_SelfHosted() {
	config.SetIsSelfHosted(true)

	defer config.SetIsSelfHosted(false)

	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	depl := s.MockDeployment(env, map[string]any{
		"BuildConfig": &buildconf.BuildConf{
			CacheDirs: []string{".next/cache", "node_modules"},
		},
	})

	message := s.deployAndReadMessage(app.App, depl.Deployment)

	s.Equal([]string{".next/cache", "node_modules"}, message.Build.CacheDirs)
}

func (s *DeploySuite) Test_Deployment_BuildCache_Cloud_PremiumTier() {
	config.SetIsStormkitCloud(true)

	defer config.SetIsStormkitCloud(false)

	usr := s.MockUser() // Mock users are premium by default.
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	depl := s.MockDeployment(env, map[string]any{
		"BuildConfig": &buildconf.BuildConf{
			CacheDirs: []string{".next/cache"},
		},
	})

	message := s.deployAndReadMessage(app.App, depl.Deployment)

	s.Equal([]string{".next/cache"}, message.Build.CacheDirs)
}

func (s *DeploySuite) Test_Deployment_BuildCache_Cloud_FreeTier() {
	config.SetIsStormkitCloud(true)

	defer config.SetIsStormkitCloud(false)

	usr := s.MockUser(map[string]any{
		"Metadata": user.UserMeta{
			SeatsPurchased: 1,
			PackageName:    config.PackageFree,
		},
	})
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	depl := s.MockDeployment(env, map[string]any{
		"BuildConfig": &buildconf.BuildConf{
			CacheDirs: []string{".next/cache"},
		},
	})

	message := s.deployAndReadMessage(app.App, depl.Deployment)

	// Free tier users on the cloud get no cache directories, which disables
	// caching in the runner.
	s.Empty(message.Build.CacheDirs)
}

func (s *DeploySuite) Test_Deployment_BuildCache_NoDirs() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	depl := s.MockDeployment(env, map[string]any{
		"BuildConfig": &buildconf.BuildConf{},
	})

	message := s.deployAndReadMessage(app.App, depl.Deployment)

	s.Empty(message.Build.CacheDirs)
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
