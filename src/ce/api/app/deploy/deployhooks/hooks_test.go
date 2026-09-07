package deployhooks_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhooks"
	"gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/testutils"
)

type HooksSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
}

func (s *HooksSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockCacheService = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCacheService
}

func (s *HooksSuite) AfterTest(_, _ string) {
	appcache.DefaultCacheService = nil
	s.conn.CloseTx()
}

// publishedCount returns how many deployments the test's data has published.
func (s *HooksSuite) publishedCount() int {
	count := 0

	row := s.conn.QueryRow(`SELECT count(*) FROM deployments_published;`)
	s.NoError(row.Scan(&count))

	return count
}

func (s *HooksSuite) TestOutboundWebhooks() {
	a := assert.New(s.T())
	ms := testutils.MockServer()
	mr := testutils.MockResponse{
		Status: 200,
		Method: shttp.MethodPost,
		Expect: func(req *http.Request) {
			b, _ := io.ReadAll(req.Body)
			post := string(b)

			a.Equal(req.Header.Get("Content-Type"), "application/json")
			a.JSONEq(post, `{"deployment_id": "1", "env_name": "production"}`)
		},
	}

	ms.NewResponse("/", &mr)
	defer ms.Close()

	depl := s.MockDeployment(nil, map[string]interface{}{
		"ExitCode":          null.NewInt(0, true),
		"PullRequestNumber": null.NewInt(0, true),
		"ShouldPublish":     true,
	})

	err := app.NewStore().InsertOutboundWebhook(context.Background(), 1, app.OutboundWebhook{
		TriggerWhen:    app.TriggerOnDeploySuccess,
		RequestURL:     ms.URL(),
		RequestMethod:  shttp.MethodPost,
		RequestPayload: null.NewString(`{ "deployment_id": "$SK_DEPLOYMENT_ID", "env_name": "$SK_ENVIRONMENT" }`, true),
		RequestHeaders: map[string]string{
			"Content-Type": "application/json",
		},
	})

	a.NoError(err)

	deployhooks.Exec(context.Background(), &deploy.Deployment{
		ID:       depl.ID,
		Env:      "production",
		Branch:   "master",
		EnvID:    s.GetEnv().ID,
		AppID:    s.GetApp().ID,
		ExitCode: null.NewInt(0, true),
	})

	a.Equal(mr.NumberOfCalls, 1)
}

func (s *HooksSuite) TestShouldNotPublish_WhenShouldPublishIsFalse() {
	env := s.MockEnv(nil, map[string]any{
		"AutoPublish": true,
	})

	depl := s.MockDeployment(env, map[string]any{
		"ExitCode":          null.NewInt(0, true),
		"PullRequestNumber": null.NewInt(0, true),
		"ShouldPublish":     false,
	})

	s.GetApp()

	deployhooks.Exec(context.Background(), depl.Deployment)

	a := assert.New(s.T())
	a.Zero(s.publishedCount(), "the deployment should not have been published")
}

func (s *HooksSuite) TestShouldNotPublish_WhenDeploymentFailed() {
	env := s.MockEnv(nil, map[string]interface{}{
		"AutoPublish": true,
	})

	depl := s.MockDeployment(env, map[string]interface{}{
		"ExitCode":          null.NewInt(1, true),
		"PullRequestNumber": null.NewInt(0, true),
		"ShouldPublish":     true,
	})

	deployhooks.Exec(context.Background(), depl.Deployment)

	a := assert.New(s.T())
	a.Zero(s.publishedCount(), "the deployment should not have been published")
}

func TestHooks(t *testing.T) {
	suite.Run(t, &HooksSuite{})
}
