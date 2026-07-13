package deploy_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type PublisherSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	mockRequest      *mocks.RequestInterface
	mockCacheService *mocks.CacheInterface
}

func (s *PublisherSuite) BeforeTest(suiteName, testName string) {
	s.conn = databasetest.InitTx(suiteName + "_" + testName)
	s.Factory = factory.New(s.conn)
	s.mockRequest = &mocks.RequestInterface{}
	s.mockCacheService = &mocks.CacheInterface{}
	shttp.DefaultRequest = s.mockRequest
	appcache.DefaultCacheService = s.mockCacheService
}

func (s *PublisherSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	s.mockCacheService = nil
	shttp.DefaultRequest = nil
	appcache.DefaultCacheService = nil
}

func (s *PublisherSuite) Test_PublishingMultiple() {
	app := s.MockApp(nil)
	env := s.MockEnv(app, nil)
	dps := s.MockDeployments(2, env)

	settings := []*deploy.PublishSettings{
		{
			EnvID:        env.ID,
			DeploymentID: dps[0].ID,
			Percentage:   25,
		},
		{
			EnvID:        env.ID,
			DeploymentID: dps[1].ID,
			Percentage:   75,
		},
	}

	s.mockCacheService.On("Reset", env.ID).Return(nil).Once()
	s.mockCacheService.On("Reset", env.ID).Return(nil).Once()

	err := deploy.Publish(context.Background(), settings)
	s.NoError(err)

	rows, err := s.conn.Query(`
		SELECT
			env_id, deployment_id, percentage_released
		FROM
			deployments_published
		ORDER BY
			percentage_released ASC;`)

	s.NoError(err)

	i := 0

	for rows.Next() {
		var envID, deploymentID types.ID
		var percentage float64
		var setting = settings[i]

		err := rows.Scan(&envID, &deploymentID, &percentage)
		s.NoError(err)
		s.Equal(envID, setting.EnvID)
		s.Equal(deploymentID, setting.DeploymentID)
		s.Equal(percentage, setting.Percentage)
		i = i + 1
	}

	// This should remove the old published environment (with ID 1)
	// and replace the existing published deployments with the new one.
	settings = []*deploy.PublishSettings{
		{
			EnvID:        env.ID,
			DeploymentID: dps[0].ID,
			Percentage:   100,
		},
	}

	s.mockCacheService.On("Reset", env.ID).Return(nil).Once()

	err = deploy.Publish(context.Background(), settings)
	s.NoError(err)

	rows, err = s.conn.Query(`
		SELECT
			env_id::text || ':' ||
			deployment_id::text || ':' ||
			percentage_released::text
		FROM deployments_published
		ORDER BY env_id ASC;`)

	s.NoError(err)

	rows.Next()
	var str string

	err = rows.Scan(&str)
	s.NoError(err)
	s.Equal(fmt.Sprintf("%d:%d:100.0", env.ID, dps[0].ID), str)
}

func (s *PublisherSuite) Test_PublishOutboundWebhooks() {
	appl := s.MockApp(nil)
	env := s.MockEnv(appl, nil)
	depl := s.MockDeployment(env)

	settings := []*deploy.PublishSettings{
		{
			EnvID:        env.ID,
			DeploymentID: depl.ID,
			Percentage:   100,
		},
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// Required for webhook dispatch
	s.mockRequest.On("Method", http.MethodPost).Return(s.mockRequest).Once()
	s.mockRequest.On("URL", "http://example.org/webhooks/publish").Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.HeadersFromMap(headers)).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(nil, nil).Once()
	s.mockRequest.On("Payload",
		fmt.Sprintf(`{ "deployment_id": "%s" }`, depl.ID.String()),
	).Return(s.mockRequest).Once()

	err := app.NewStore().InsertOutboundWebhook(context.Background(), appl.ID, app.OutboundWebhook{
		TriggerWhen:    app.TriggerOnPublish,
		RequestURL:     "http://example.org/webhooks/publish",
		RequestMethod:  shttp.MethodPost,
		RequestPayload: null.NewString(`{ "deployment_id": "$SK_DEPLOYMENT_ID" }`, true),
		RequestHeaders: headers,
	})

	s.mockCacheService.On("Reset", env.ID).Return(nil)

	s.NoError(err)
	s.Nil(deploy.Publish(context.Background(), settings))
}

func (s *PublisherSuite) Test_AutoPublish() {
	app := s.MockApp(nil)
	env := s.MockEnv(app, nil)
	depl := s.MockDeployment(env, map[string]any{
		"ExitCode":          null.NewInt(0, true),
		"PullRequestNumber": null.NewInt(0, true),
		"ShouldPublish":     true,
	})

	// Should reset the cache
	s.mockCacheService.On("Reset", env.ID).Return(nil).Once()

	err := deploy.AutoPublishIfNecessary(context.Background(), depl.Deployment)
	s.NoError(err)

	depls, err := deploy.NewStore().MyDeployments(context.Background(), &deploy.DeploymentsQueryFilters{
		EnvID:     env.ID,
		Published: aws.Bool(true),
	})

	s.NoError(err)
	s.Len(depls, 1)
	s.Equal(depl.ID, depls[0].ID)
}

func (s *PublisherSuite) Test_AutoPublish_Audit_SelfHosted() {
	appl := s.MockApp(nil)
	env := s.MockEnv(appl, nil)
	depl := s.MockDeployment(env, map[string]any{
		"ExitCode":          null.NewInt(0, true),
		"PullRequestNumber": null.NewInt(0, true),
		"ShouldPublish":     true,
	})

	admin.SetMockLicense()
	defer func() { admin.ResetMockLicense() }()

	s.mockCacheService.On("Reset", env.ID).Return(nil).Once()

	err := deploy.AutoPublishIfNecessary(context.Background(), depl.Deployment)
	s.NoError(err)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		AppID: appl.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal("UPDATE:DEPLOYMENT", audits[0].Action)
	s.Equal(appl.ID, audits[0].AppID)
	s.Equal(env.ID, audits[0].EnvID)
	s.Equal(depl.ID.String(), audits[0].Diff.New.DeploymentID)
	s.True(*audits[0].Diff.New.AutoPublished)
}

func (s *PublisherSuite) Test_AutoPublish_Audit_Cloud() {
	appl := s.MockApp(nil)
	env := s.MockEnv(appl, nil)
	depl := s.MockDeployment(env, map[string]any{
		"ExitCode":          null.NewInt(0, true),
		"PullRequestNumber": null.NewInt(0, true),
		"ShouldPublish":     true,
	})

	config.SetIsStormkitCloud(true)
	defer config.SetIsStormkitCloud(false)

	s.mockCacheService.On("Reset", env.ID).Return(nil).Once()

	err := deploy.AutoPublishIfNecessary(context.Background(), depl.Deployment)
	s.NoError(err)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		AppID: appl.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal("UPDATE:DEPLOYMENT", audits[0].Action)
	s.Equal(appl.ID, audits[0].AppID)
	s.Equal(env.ID, audits[0].EnvID)
	s.Equal(depl.ID.String(), audits[0].Diff.New.DeploymentID)
	s.True(*audits[0].Diff.New.AutoPublished)
}

func TestPublisher(t *testing.T) {
	suite.Run(t, &PublisherSuite{})
}
