package publicapiv1_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerRedirectsGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerRedirectsGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerRedirectsGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerRedirectsGetSuite) TestSuccess() {
	reds := []redirects.Redirect{
		{From: "/path", To: "/new-path", Status: http.StatusFound},
		{From: "*", To: "/index.html"},
	}

	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"Data": &buildconf.BuildConf{
			Redirects: reds,
		},
	})

	key := s.MockAPIKey(nil, env)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/redirects",
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	jsonVal := `{
		"redirects": [
			{ "from": "/path", "to": "/new-path", "status": 302 },
			{ "from": "*", "to": "/index.html" }
		]
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(string(jsonVal), response.String())
}

func TestHandlerRedirectsGet(t *testing.T) {
	suite.Run(t, &HandlerRedirectsGetSuite{})
}
