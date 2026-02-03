package adminhandlers_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type HandlerLicenseSetSuite struct {
	suite.Suite
	*factory.Factory

	conn        databasetest.TestDB
	mockRequest *mocks.RequestInterface
}

func (s *HandlerLicenseSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockRequest = &mocks.RequestInterface{}
	shttp.DefaultRequest = s.mockRequest
}

func (s *HandlerLicenseSetSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	shttp.DefaultRequest = nil
}

func (s *HandlerLicenseSetSuite) mockResponse(token string, responseStatus int, responseBody string, calledTimes int) {
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")

	s.mockRequest.On("URL", fmt.Sprintf("https://api.stormkit.io/v1/license/check?token=%s", token)).Return(s.mockRequest).Times(calledTimes)
	s.mockRequest.On("Method", http.MethodGet).Return(s.mockRequest).Times(calledTimes)
	s.mockRequest.On("Headers", headers).Return(s.mockRequest).Times(calledTimes)
	s.mockRequest.On("WithExponentialBackoff", 5*time.Minute, 10).Return(s.mockRequest).Times(calledTimes)
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: responseStatus,
			Body:       io.NopCloser(strings.NewReader(responseBody)),
		},
	}, nil).Times(calledTimes)
}

func (s *HandlerLicenseSetSuite) Test_Success_WithValidLicense() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})
	key := "valid-license-key"

	// Twice because we call it once to validate the license and once after cache is updated
	s.mockResponse(key, http.StatusOK, `{ "license": {"seats":7, "ultimate": true, "premium": false, "version":"2025-09-26"} }`, 2)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/license",
		map[string]any{
			"key": key,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{ "seats":7, "ultimate": true, "premium": false, "edition": "enterprise" }`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())

	// The config should have the new license key
	config, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.Equal(key, config.LicenseConfig.Key)
}

func (s *HandlerLicenseSetSuite) Test_Success_RemoveLicense() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	// First set a license
	validLicense := admin.NewLicense(admin.NewLicenseArgs{
		Seats:   50,
		Premium: true,
		UserID:  usr.ID,
	})

	// Set initial license
	config := admin.InstanceConfig{
		LicenseConfig: &admin.LicenseConfig{
			Key: validLicense.Key,
		},
	}

	s.mockResponse(validLicense.Key, http.StatusOK, `{ "license": {"seats":50,"version":"2025-09-26"} }`, 1)

	err := admin.Store().UpsertConfig(context.Background(), config)
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/license",
		map[string]any{
			"key": "",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{ "seats": -1, "premium": false, "ultimate": false, "edition": "community" }`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())

	// The config should have the new license key
	config, err = admin.Store().Config(context.Background())
	s.NoError(err)
	s.Equal("", config.LicenseConfig.Key)
}

func (s *HandlerLicenseSetSuite) Test_InvalidLicense() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	s.mockResponse("invalid-key", http.StatusUnauthorized, ``, 1)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/license",
		map[string]any{
			"key": "invalid-key",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{ "error": "license is either invalid or no longer active" }`, response.String())
}

func (s *HandlerLicenseSetSuite) Test_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/license",
		map[string]any{
			"key": "some-license-key",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerLicenseSetSuite(t *testing.T) {
	suite.Run(t, &HandlerLicenseSetSuite{})
}
