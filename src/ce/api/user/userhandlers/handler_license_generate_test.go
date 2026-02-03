package userhandlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

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

type HandlerLicenseGenerateSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerLicenseGenerateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	config.SetIsStormkitCloud(true)
}

func (s *HandlerLicenseGenerateSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	config.SetIsStormkitCloud(false)
}

func (s *HandlerLicenseGenerateSuite) Test_Success() {
	usr := s.MockUser(map[string]any{
		"Metadata": user.UserMeta{
			SeatsPurchased: 10,
			PackageName:    config.PackagePremium,
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/license",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Parse response and validate structure
	var responseData map[string]interface{}
	s.NoError(json.Unmarshal(response.Byte(), &responseData))

	// Check that key is returned
	s.Contains(responseData, "key")
	key, ok := responseData["key"].(string)
	s.True(ok, "Key should be a string")
	s.NotEmpty(key, "Key should not be empty")
}

func (s *HandlerLicenseGenerateSuite) Test_NoSeatsPurchased() {
	usr := s.MockUser(map[string]any{
		"Metadata": user.UserMeta{
			SeatsPurchased: 0,
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/license",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"User has not purchased any seats"}`, response.String())
}

func (s *HandlerLicenseGenerateSuite) Test_NegativeSeats() {
	usr := s.MockUser(map[string]any{
		"Metadata": user.UserMeta{
			SeatsPurchased: -5,
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/license",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"User has not purchased any seats"}`, response.String())
}

func (s *HandlerLicenseGenerateSuite) Test_ReplacesExistingLicense() {
	usr := s.MockUser(map[string]any{
		"Metadata": user.UserMeta{
			SeatsPurchased: 25,
			PackageName:    config.PackagePremium,
		},
	})

	// Create an existing license first
	l, err := user.NewStore().GenerateSelfHostedLicense(
		context.Background(),
		usr.Metadata.SeatsPurchased,
		usr.ID,
		config.PackagePremium,
		nil,
	)

	s.NoError(err)
	s.NotNil(l)

	// Generate new license
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/license",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Parse response
	var responseData map[string]any
	s.NoError(json.Unmarshal(response.Byte(), &responseData))

	// Check that new key is returned
	s.Contains(responseData, "key")
	newKey, ok := responseData["key"].(string)
	s.True(ok, "Key should be a string")
	s.NotEmpty(newKey, "Key should not be empty")

	// Verify the license was updated with new seats count
	license, err := user.NewStore().LicenseByUserID(context.Background(), usr.ID)
	s.NoError(err)
	s.NotNil(license)
	s.Equal(25, license.Seats) // Should have the new seat count
}

func (s *HandlerLicenseGenerateSuite) Test_NoAuth() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/user/license",
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerLicenseGenerateSuite(t *testing.T) {
	suite.Run(t, &HandlerLicenseGenerateSuite{})
}
