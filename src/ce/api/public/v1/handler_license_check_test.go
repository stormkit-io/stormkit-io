package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerLicenseCheckSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerLicenseCheckSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	config.SetIsStormkitCloud(true)
}

func (s *HandlerLicenseCheckSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerLicenseCheckSuite) Test_Success_MethodGET() {
	ctx := context.Background()
	store := user.NewStore()

	license, err := store.GenerateSelfHostedLicense(ctx, 5, config.PackagePremium, nil)
	s.NoError(err)
	s.NotNil(license)

	response := shttptest.Request(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/license/check?token=%s", license.Key),
		nil,
	)

	str := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.Require().NotEmpty(str, "response body should not be empty")
	s.JSONEq(`{"license": { "seats": 5, "version": "2025-09-26", "premium": true, "ultimate": false }}`, str)
}

func (s *HandlerLicenseCheckSuite) Test_Success_MethodPOST() {
	ctx := context.Background()
	store := user.NewStore()

	license, err := store.GenerateSelfHostedLicense(ctx, 5, config.PackagePremium, nil)
	s.NoError(err)
	s.NotNil(license)

	response := shttptest.Request(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/license/check",
		map[string]any{
			"token": license.Key,
		},
	)

	str := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.Require().NotEmpty(str, "response body should not be empty")
	s.JSONEq(`{"license": { "seats": 5, "version": "2025-09-26", "premium": true, "ultimate": false }}`, str)
}

func (s *HandlerLicenseCheckSuite) Test_InvalidLicense() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/license/check?token=%s", "1:234-5678-90ab-cdef-1234-5678-90ab-cdef"),
		nil,
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerLicenseCheckSuite) Test_MissingToken_GET() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/license/check",
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func (s *HandlerLicenseCheckSuite) Test_MissingToken_POST() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/license/check",
		map[string]any{},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerLicenseCheckSuite(t *testing.T) {
	suite.Run(t, &HandlerLicenseCheckSuite{})
}
