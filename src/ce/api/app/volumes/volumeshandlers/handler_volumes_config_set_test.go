package volumeshandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes/volumeshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type HandlerVolumesConfigSetSuite struct {
	suite.Suite
	*factory.Factory
	conn    databasetest.TestDB
	service *mocks.MicroServiceInterface
}

func (s *HandlerVolumesConfigSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.service = &mocks.MicroServiceInterface{}
	admin.ResetCache(context.Background())
	rediscache.DefaultService = s.service
	admin.SetMockLicense()
}

func (s *HandlerVolumesConfigSetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	rediscache.DefaultService = nil
	admin.ResetMockLicense()
}

func (s *HandlerVolumesConfigSetSuite) Test_Success_IsAdmin() {
	usr := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	s.service.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/volumes/config",
		map[string]any{
			"mountType": "filesys",
			"rootPath":  "/test",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	config, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.Equal(admin.VolumesConfig{MountType: volumes.FileSys, RootPath: "/test"}, *config.VolumesConfig)
}

func (s *HandlerVolumesConfigSetSuite) Test_Success_AWS() {
	usr := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	s.service.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/volumes/config",
		map[string]any{
			"mountType":  "aws:s3",
			"accessKey":  "hello",
			"secretKey":  "world",
			"bucketName": "my-bucket",
			"region":     "eu-central-1",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	config, err := admin.Store().Config(context.Background())

	s.NoError(err)
	s.Equal(admin.VolumesConfig{
		MountType:  volumes.AWSS3,
		AccessKey:  "hello",
		SecretKey:  "world",
		Region:     "eu-central-1",
		BucketName: "my-bucket",
	}, admin.VolumesConfig{
		MountType:  config.VolumesConfig.MountType,
		AccessKey:  config.VolumesConfig.AccessKey,
		SecretKey:  config.VolumesConfig.SecretKey,
		Region:     config.VolumesConfig.Region,
		BucketName: config.VolumesConfig.BucketName,
	})
}

func (s *HandlerVolumesConfigSetSuite) Test_Fail_InvalidMountType() {
	usr := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/volumes/config",
		map[string]any{
			"mountType": "aws",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{ "error": "Invalid mount type given. Valid values are: aws:s3, filesys" }`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerVolumesConfigSetSuite) Test_Fail_NotAdmin() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/volumes/config",
		map[string]any{
			"mountType": "filesys",
			"rootPath":  "/test",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)

	config, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.True(config.IsEmpty())
}

func TestHandlerVolumesConfigSet(t *testing.T) {
	suite.Run(t, &HandlerVolumesConfigSetSuite{})
}
