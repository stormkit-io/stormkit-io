package adminhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerAdminMailerGetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAdminMailerGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAdminMailerGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetCache(context.Background())
}

func (s *HandlerAdminMailerGetSuite) get() shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/mailer",
		nil,
		map[string]string{"Authorization": usertest.Authorization(s.MockUser(map[string]any{"IsAdmin": true}).ID)},
	)
}

func (s *HandlerAdminMailerGetSuite) Test_NotConfigured() {
	response := s.get()

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"mailer":null}`, response.String())
}

func (s *HandlerAdminMailerGetSuite) Test_MasksPassword() {
	cnf := admin.MustConfig().Clone()
	cnf.MailerConfig = &admin.MailerConfig{
		Host:     "smtp.instance.test",
		Username: "instance-user",
		Password: "instance-secret",
		From:     "noreply@instance.test",
	}

	s.Require().NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := s.get()

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"mailer":{
		"host": "smtp.instance.test",
		"port": "",
		"username": "instance-user",
		"password": "****-****-****-****",
		"from": "noreply@instance.test"
	}}`, response.String())
}

func TestHandlerAdminMailerGetSuite(t *testing.T) {
	suite.Run(t, &HandlerAdminMailerGetSuite{})
}
