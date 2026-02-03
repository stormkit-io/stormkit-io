package admin_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type LicenseSuite struct {
	suite.Suite
	*factory.Factory
	conn        databasetest.TestDB
	mockRequest *mocks.RequestInterface
}

func NewMockLicense() *admin.License {
	return &admin.License{
		Seats:   1,
		Key:     utils.RandomToken(128),
		Version: admin.LicenseVersion20240610,
	}
}

func (s *LicenseSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.ResetMockLicense()
	admin.Store().DeleteConfig(context.Background())
	config.SetIsSelfHosted(true)
	os.Unsetenv("STORMKIT_LICENSE")
	s.mockRequest = &mocks.RequestInterface{}
	shttp.DefaultRequest = s.mockRequest
}

func (s *LicenseSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *LicenseSuite) Test_FreeLicense() {
	license := admin.CurrentLicense()
	s.NotNil(license)
	s.Equal(admin.MaximumFreeSeats, license.Seats)
	s.False(license.IsEnterprise())
}

func (s *LicenseSuite) Test_FetchingLicenseFromDB_Success() {
	license := admin.NewLicense(admin.NewLicenseArgs{
		Seats:   5,
		Premium: true,
		Key:     "abcd-efgh-ijkl-mnop",
	})

	cnf, err := admin.Store().Config(context.Background())
	s.NoError(err)

	cnf.LicenseConfig = &admin.LicenseConfig{
		Key: license.Key,
	}

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")

	s.mockRequest.On("URL", "https://api.stormkit.io/v1/license/check?token=abcd-efgh-ijkl-mnop").Return(s.mockRequest).Once()
	s.mockRequest.On("Method", http.MethodGet).Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", headers).Return(s.mockRequest).Once()
	s.mockRequest.On("WithExponentialBackoff", 5*time.Minute, 10).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{ "license": {"seats":7, "premium": true, "version":"2025-09-26"} }`)),
		},
	}, nil).Once()

	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	l := admin.CurrentLicense()
	s.NotNil(l)
	s.Equal(7, l.Seats)
	s.True(l.IsEnterprise())

	cnf, err = admin.Store().Config(context.Background())
	s.NoError(err)
	s.Equal("abcd-efgh-ijkl-mnop", cnf.LicenseConfig.Key)
}

func (s *LicenseSuite) Test_FetchingLicenseFromDB_ExpiredLicense() {
	license := admin.NewLicense(admin.NewLicenseArgs{
		Seats: 5,
		Key:   "abcd-efgh-ijkl-mnop",
	})

	cnf, err := admin.Store().Config(context.Background())
	s.NoError(err)

	cnf.LicenseConfig = &admin.LicenseConfig{
		Key: license.Key,
	}

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")

	s.mockRequest.On("URL", "https://api.stormkit.io/v1/license/check?token=abcd-efgh-ijkl-mnop").Return(s.mockRequest).Once()
	s.mockRequest.On("Method", http.MethodGet).Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", headers).Return(s.mockRequest).Once()
	s.mockRequest.On("WithExponentialBackoff", 5*time.Minute, 10).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{ "license": {} }`)),
		},
	}, nil).Once()

	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	l := admin.CurrentLicense()
	s.NotNil(l)
	s.Equal(admin.MaximumFreeSeats, l.Seats)
	s.False(l.IsEnterprise())

	cnf, err = admin.Store().Config(context.Background())
	s.NoError(err)
	s.Equal(license.Key, cnf.LicenseConfig.Key)
}

func TestLicenseSuite(t *testing.T) {
	suite.Run(t, &LicenseSuite{})
}
