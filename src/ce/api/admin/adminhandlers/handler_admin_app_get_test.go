package adminhandlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type HandlerAdminAppGetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAdminAppGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	config.SetIsStormkitCloud(true)
}

func (s *HandlerAdminAppGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsStormkitCloud(false)
}

func (s *HandlerAdminAppGetSuite) Test_Get_ByDisplayName() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})
	app := s.MockApp(usr)

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/admin/cloud/app?url=http://%s.stormkit:8888", app.DisplayName),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]map[string]any{}

	s.Equal(http.StatusOK, resp.Code)
	s.NoError(json.Unmarshal(resp.Byte(), &data))
	s.Equal(app.DisplayName, data["app"]["displayName"])
	s.Equal(app.Repo, data["app"]["repo"])
	s.Equal(app.ID.String(), data["app"]["id"])
	s.Equal(usr.ID.String(), data["user"]["id"])
}

func (s *HandlerAdminAppGetSuite) Test_Get_ByDomainName() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	domain := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "www.stormkit.io",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom("my-custom-token"),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/cloud/app?url=https://www.stormkit.io",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]map[string]any{}

	s.Equal(http.StatusOK, resp.Code)
	s.NoError(json.Unmarshal(resp.Byte(), &data))
	s.Equal(app.DisplayName, data["app"]["displayName"])
	s.Equal(app.Repo, data["app"]["repo"])
	s.Equal(app.ID.String(), data["app"]["id"])
}

func (s *HandlerAdminAppGetSuite) Test_Get_Unauthorized_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/cloud/app",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, resp.Code)
}

func TestHandlerAdminAppGetSuite(t *testing.T) {
	suite.Run(t, &HandlerAdminAppGetSuite{})
}
