package redirectshandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects/redirectshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerPlaygroundSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerPlaygroundSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerPlaygroundSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerPlaygroundSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	// This is required to fetch the app conf
	s.MockDeployment(env, map[string]any{
		"Published": deploy.PublishedInfo{
			{EnvID: env.ID, Percentage: 100},
		},
	})

	domain := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "www.stormkit.io",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(redirectshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/redirects/playground",
		map[string]any{
			"appId":   app.ID.String(),
			"envId":   env.ID.String(),
			"address": "https://www.stormkit.io/old-docs",
			"redirects": []map[string]any{
				{"from": "/old-docs", "to": "/docs", "status": 301},
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"match": true,
		"against": "https://www.stormkit.io/old-docs",
		"pattern": "^/old-docs$",
		"proxy": false,
		"redirect": "https://www.stormkit.io/docs",
		"rewrite": "",
		"status": 301
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerPlayground(t *testing.T) {
	suite.Run(t, &HandlerPlaygroundSuite{})
}
