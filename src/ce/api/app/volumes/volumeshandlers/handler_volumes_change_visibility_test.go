package volumeshandlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes/volumeshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerVolumesChangeVisibilitySuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerVolumesChangeVisibilitySuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerVolumesChangeVisibilitySuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerVolumesChangeVisibilitySuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	ctx := context.Background()

	t, err := time.Parse(time.DateTime, "2024-04-06 15:45:30")
	s.NoError(err)

	files := []*volumes.File{
		{Name: "test.txt", Size: 13, Path: "", IsPublic: false, CreatedAt: utils.Unix{Valid: true, Time: t}},
	}

	s.NoError(volumes.Store().Insert(ctx, files, env.ID))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/volumes/visibility",
		map[string]string{
			"appId":      app.ID.String(),
			"envId":      env.ID.String(),
			"fileId":     files[0].ID.String(),
			"visibility": "public",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	file, err := volumes.Store().FileByID(ctx, files[0].ID)

	s.NoError(err)
	s.Equal(files[0].Name, file.Name)
	s.Equal(files[0].Size, file.Size)
	s.True(file.IsPublic)
}

func (s *HandlerVolumesChangeVisibilitySuite) Test_InvalidParameter() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/volumes/visibility",
		map[string]string{
			"appId":      app.ID.String(),
			"envId":      env.ID.String(),
			"fileId":     "1",
			"visibility": "invalid",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{ "error": "Invalid visibility provided: can be one of public or private." }`, response.String())
}

func TestHandlerVolumesChangeVisibility(t *testing.T) {
	suite.Run(t, &HandlerVolumesChangeVisibilitySuite{})
}
