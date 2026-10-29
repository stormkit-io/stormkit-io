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

type HandlerAdminMailerDeleteSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAdminMailerDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAdminMailerDeleteSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetCache(context.Background())
}

// Test_Success verifies that the mailer is removed and the rest of the
// instance config is kept.
func (s *HandlerAdminMailerDeleteSuite) Test_Success() {
	cnf := admin.MustConfig().Clone()
	cnf.DomainConfig = &admin.DomainConfig{API: "https://api.instance.test"}
	cnf.MailerConfig = &admin.MailerConfig{Host: "smtp.instance.test", Username: "u", Password: "p", From: "noreply@instance.test"}

	s.Require().NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		"/admin/mailer",
		nil,
		map[string]string{"Authorization": usertest.Authorization(s.MockUser(map[string]any{"IsAdmin": true}).ID)},
	)

	s.Equal(http.StatusOK, response.Code)

	admin.ResetCache(context.Background())

	stored, err := admin.Store().Config(context.Background())
	s.Require().NoError(err)
	s.Nil(stored.MailerConfig)
	s.Equal("https://api.instance.test", stored.DomainConfig.API)
}

func TestHandlerAdminMailerDeleteSuite(t *testing.T) {
	suite.Run(t, &HandlerAdminMailerDeleteSuite{})
}
