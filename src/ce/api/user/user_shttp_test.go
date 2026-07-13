package user_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type UserSHTTPSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *UserSHTTPSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	cfg := admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				Whitelist:  []string{"stormkit.io", "example.org"},
				SignUpMode: admin.SIGNUP_MODE_WAITLIST,
			},
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cfg))

	// Enable self-hosted mode for testing enterprise features
	admin.SetMockLicense()
}

func (s *UserSHTTPSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsSelfHosted(false)
	admin.ResetMockLicense()
}

func (s *UserSHTTPSuite) Test_WithAPIKey_Success() {
	usr := s.MockUser()
	tkn := &apikey.Token{
		Scope:  apikey.SCOPE_USER,
		UserID: usr.ID,
		Value:  apikey.GenerateTokenValue(),
	}

	apikey.NewStore().AddAPIKey(context.Background(), tkn)

	headers := make(http.Header)
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", tkn.Value))

	res := user.WithAPIKey(func(rc *user.RequestContext) *shttp.Response {
		return shttp.OK()
	})(&shttp.RequestContext{
		Request: &http.Request{
			Header: headers,
		},
	})

	s.Equal(res.Status, http.StatusOK)
}

func (s *UserSHTTPSuite) Test_WithAPIKey_Forbidden() {
	headers := make(http.Header)

	res := user.WithAPIKey(func(rc *user.RequestContext) *shttp.Response {
		return shttp.OK()
	})(&shttp.RequestContext{
		Request: &http.Request{
			Header: headers,
		},
	})

	s.Equal(res.Status, http.StatusForbidden)
}

func (s *UserSHTTPSuite) Test_WithAuth_Success() {
	usr := s.MockUser()

	headers := make(http.Header)
	headers.Set("Authorization", usertest.Authorization(usr.ID))

	res := user.WithAuth(func(rc *user.RequestContext) *shttp.Response {
		return shttp.OK()
	})(&shttp.RequestContext{
		Request: &http.Request{
			Header: headers,
		},
	})

	s.Equal(res.Status, http.StatusOK)
}

func (s *UserSHTTPSuite) Test_WithAuth_NotApproved() {
	usr := s.MockUser(map[string]any{
		"IsApproved": null.BoolFrom(false),
	})

	headers := make(http.Header)
	headers.Set("Authorization", usertest.Authorization(usr.ID))

	res := user.WithAuth(func(rc *user.RequestContext) *shttp.Response {
		return shttp.OK()
	})(&shttp.RequestContext{
		Request: &http.Request{
			Header: headers,
		},
	})

	s.Equal(http.StatusForbidden, res.Status)
	s.JSONEq(`{ "error": "Your account is not approved by an administrator. You cannot access this resource." }`, res.String())
}

func (s *UserSHTTPSuite) Test_WithAuth_Approved() {
	usr := s.MockUser(map[string]any{
		"IsApproved": null.BoolFrom(true),
	})

	headers := make(http.Header)
	headers.Set("Authorization", usertest.Authorization(usr.ID))

	res := user.WithAuth(func(rc *user.RequestContext) *shttp.Response {
		return shttp.OK()
	})(&shttp.RequestContext{
		Request: &http.Request{
			Header: headers,
		},
	})

	s.Equal(res.Status, http.StatusOK)
}

func TestUserSHTTPSuite(t *testing.T) {
	suite.Run(t, &UserSHTTPSuite{})
}
