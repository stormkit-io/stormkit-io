package userhandlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/userhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type UserSessionSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *UserSessionSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.CachedLicense = &admin.License{}
}

func (s *UserSessionSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *UserSessionSuite) Test_Success_SelfHosted() {
	config.SetIsSelfHosted(true)
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/user",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"accounts": [
			{ "provider": "github", "url": "https://api.github.com/users/dlorenzo", "displayName": "dlorenzo", "hasPersonalAccessToken": false },
			{ "provider": "bitbucket", "url": "https://bitbucket.org/dlorenzo", "displayName": "dlorenzo", "hasPersonalAccessToken": false },
			{ "provider": "gitlab", "url":"https://gitlab.com/dlorenzo", "displayName": "dlorenzo", "hasPersonalAccessToken": true }
		],
		"user": {
			"avatar": "https://avatars3.githubusercontent.com/u/55663230?v=4",
			"displayName": "dlorenzo",
			"fullName": "David Lorenzo",
			"id": "%s",
			"email": "%s",
			"memberSince": 1551193200,
			"package": {
				"id": "premium",
				"seats": 1
			}
		}
	}`, usr.ID.String(), usr.PrimaryEmail())

	s.Equal(response.Code, http.StatusOK)
	s.JSONEq(expected, response.String())
}

func (s *UserSessionSuite) Test_Success_Cloud() {
	config.SetIsStormkitCloud(true)
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/user",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	var data map[string]any

	s.Equal(response.Code, http.StatusOK)
	s.NoError(json.Unmarshal(response.Byte(), &data))
	s.Equal(data["user"].(map[string]any)["id"], usr.ID.String())

	expected := `{
		"max": {
			"bandwidthInBytes": 1e+12,
			"storageInBytes": 1e+12,
			"buildMinutes": 1000,
			"functionInvocations": 1500000
	    },
		"used": {
			"bandwidthInBytes": 0,
			"buildMinutes": 0,
			"functionInvocations": 0,
			"storageInBytes": 0
		}
	}`

	received, err := json.Marshal(data["metrics"].(map[string]any))
	s.NoError(err)
	s.JSONEq(expected, string(received))
}

func (s *UserSessionSuite) Test_Success_Cloud_FreeUser() {
	config.SetIsStormkitCloud(true)
	usr := s.MockUser(map[string]any{
		"Metadata": user.UserMeta{
			PackageName: config.PackageFree,
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/user",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	var data map[string]any

	s.Equal(response.Code, http.StatusOK)
	s.NoError(json.Unmarshal(response.Byte(), &data))
	s.Equal(data["user"].(map[string]any)["id"], usr.ID.String())

	expected := `{
		"max": {
			"bandwidthInBytes": 1e+11,
			"storageInBytes": 1e+11,
			"buildMinutes": 300,
			"functionInvocations": 500000
	    },
		"used": {
			"bandwidthInBytes": 0,
			"buildMinutes": 0,
			"functionInvocations": 0,
			"storageInBytes": 0
		}
	}`

	received, err := json.Marshal(data["metrics"].(map[string]any))
	s.NoError(err)
	s.JSONEq(expected, string(received))
}

func (s *UserSessionSuite) Test_NotAllowedBecauseExpired() {
	req := &user.RequestContext{}
	tkn, _ := req.JWT(map[string]any{
		"issued": time.Now().Add(-30 * time.Hour).Unix(),
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/user",
		nil,
		map[string]string{
			"Authorization": tkn,
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *UserSessionSuite) Test_NotAllowedBecauseNoToken() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/user",
		nil,
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestUserSession(t *testing.T) {
	suite.Run(t, &UserSessionSuite{})
}
