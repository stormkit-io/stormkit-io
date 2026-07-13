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
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerVolumesConfigGetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerVolumesConfigGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerVolumesConfigGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerVolumesConfigGetSuite) Test_Success_IsAdmin_NoConfig() {
	usr := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/volumes/config",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"config": null
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerVolumesConfigGetSuite) Test_Success_IsAdmin_WithConfig() {
	usr := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	cfg := admin.InstanceConfig{
		VolumesConfig: &admin.VolumesConfig{
			MountType: volumes.FileSys,
			RootPath:  "/test",
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cfg))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/volumes/config",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"config": {
			"mountType": "filesys",
			"rootPath": "/test"
		}
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerVolumesConfigGetSuite) Test_Success_NotAdmin_WithConfig() {
	usr := s.MockUser()
	cfg := admin.InstanceConfig{
		VolumesConfig: &admin.VolumesConfig{
			MountType: volumes.FileSys,
			RootPath:  "/test",
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cfg))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/volumes/config",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{ "config": true }`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerVolumesConfigGet(t *testing.T) {
	suite.Run(t, &HandlerVolumesConfigGetSuite{})
}
