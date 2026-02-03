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

func (s *HandlerLicenseCheckSuite) Test_Success() {
	usr := s.MockUser()
	ctx := context.Background()
	store := user.NewStore()

	license, err := store.GenerateSelfHostedLicense(ctx, 5, usr.ID, config.PackagePremium, nil)
	s.NoError(err)
	s.NotNil(license)

	license, err = store.LicenseByUserID(context.Background(), usr.ID)
	s.NoError(err)
	s.NotNil(license)

	response := shttptest.Request(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/license/check?token=%s", license.Token()),
		nil,
	)

	str := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.Require().NotEmpty(str, "response body should not be empty")
	s.JSONEq(`{"license": { "seats": 5, "version": "2025-09-26", "premium": true, "ultimate": false }}`, str)

	// Let's update the subscription to free and retest
	s.NoError(store.UpdateSubscription(ctx, usr.ID, user.UserMeta{
		StripeCustomerID: "cus_test123",
		SeatsPurchased:   0,
		PackageName:      config.PackageFree,
	}))

	license, err = store.LicenseByUserID(context.Background(), usr.ID)
	s.NoError(err)
	s.Nil(license)
}

func (s *HandlerLicenseCheckSuite) Test_InvalidLicenseFormat() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/license/check?token=%s", "my-token"),
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
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

func TestHandlerLicenseCheckSuite(t *testing.T) {
	suite.Run(t, &HandlerLicenseCheckSuite{})
}
