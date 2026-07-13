package deployhandlers

import (
	"fmt"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerManifestSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerManifestSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerManifestSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerManifestSuite) TestSuccess() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr)
	env := s.Factory.MockEnv(app)
	dep := s.Factory.MockDeployment(env, nil)

	config.Get().Deployer.Service = "github"

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%d/manifest/%d", app.ID, dep.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	res := response.String()
	expected := `{
		"manifest": {
			"cdnFiles": [
				{ "fileName": "index", "headers": { "Keep-Alive": "30" } },
                { "fileName": "about", "headers": { "Accept-Encoding": "None" } }
            ]
        }
    }`

	s.JSONEq(expected, res)
}

func TestQuery(t *testing.T) {
	suite.Run(t, &HandlerManifestSuite{})
}
