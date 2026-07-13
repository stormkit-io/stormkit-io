package userhandlers_test

import (
	"net/http"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/userhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type UserSharedSessionSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *UserSharedSessionSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *UserSharedSessionSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *UserSharedSessionSuite) Test_Success_DefaultsToOneHour() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/shared-session",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()
	token, ok := body["token"].(string)
	s.True(ok)
	s.NotEmpty(token)

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: token})
	s.NotNil(claims)
	s.Equal(usr.ID.String(), claims["uid"])

	exp, _ := claims["exp"].(float64)
	s.InDelta(time.Now().Add(time.Hour).Unix(), int64(exp), 5)
}

func (s *UserSharedSessionSuite) Test_Success_CustomHours() {
	admin.SetMockLicense()
	defer admin.ResetMockLicense()

	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/shared-session",
		map[string]any{"hours": 6},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: response.Map()["token"].(string)})
	s.NotNil(claims)

	exp, _ := claims["exp"].(float64)
	s.InDelta(time.Now().Add(6*time.Hour).Unix(), int64(exp), 5)
}

func (s *UserSharedSessionSuite) Test_OutOfRangeHoursClampsToOne() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/shared-session",
		map[string]any{"hours": 99},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: response.Map()["token"].(string)})
	s.NotNil(claims)

	exp, _ := claims["exp"].(float64)
	s.InDelta(time.Now().Add(time.Hour).Unix(), int64(exp), 5)
}

func (s *UserSharedSessionSuite) Test_TokenExpiresAfterOneHour() {
	usr := s.MockUser()

	// Generate a shared session token with exp set 1h in the past.
	expired, err := user.JWT(jwt.MapClaims{
		"uid": usr.ID,
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/user",
		nil,
		map[string]string{
			"Authorization": "Bearer " + expired,
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *UserSharedSessionSuite) Test_EnterpriseLicenseClampsTo24h() {
	admin.SetMockLicense()
	defer admin.ResetMockLicense()

	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/shared-session",
		map[string]any{"hours": 24},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: response.Map()["token"].(string)})
	s.NotNil(claims)

	exp, _ := claims["exp"].(float64)
	s.InDelta(time.Now().Add(24*time.Hour).Unix(), int64(exp), 5)
}

func (s *UserSharedSessionSuite) Test_NonEnterpriseIgnoresHoursAndClampsToOne() {
	admin.CachedLicense = &admin.License{}
	defer func() { admin.CachedLicense = nil }()

	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/shared-session",
		map[string]any{"hours": 6},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: response.Map()["token"].(string)})
	s.NotNil(claims)

	exp, _ := claims["exp"].(float64)
	s.InDelta(time.Now().Add(time.Hour).Unix(), int64(exp), 5)
}

func (s *UserSharedSessionSuite) Test_NotAllowedWithoutAuth() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/shared-session",
		nil,
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestUserSharedSession(t *testing.T) {
	suite.Run(t, &UserSharedSessionSuite{})
}
