package volumeshandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes/volumeshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerVolumesDownloadSuite struct {
	suite.Suite
	*factory.Factory
	conn   databasetest.TestDB
	tmpdir string
}

func (s *HandlerVolumesDownloadSuite) SetupSuite() {
	tmpDir, err := os.MkdirTemp("", "tmp-volumes-")
	s.tmpdir = tmpDir
	s.NoError(err)
}

func (s *HandlerVolumesDownloadSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerVolumesDownloadSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerVolumesDownloadSuite) TearDownSuite() {
	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}
}

func (s *HandlerVolumesDownloadSuite) prepareFile(authToken string, appID, envID types.ID) (string, error) {
	ctx := context.Background()
	cfg := admin.InstanceConfig{
		VolumesConfig: &admin.VolumesConfig{
			MountType: volumes.FileSys,
			RootPath:  s.tmpdir,
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cfg))

	t, err := time.Parse(time.DateTime, "2024-04-06 15:45:30")
	s.NoError(err)

	content := []byte("Hello world!")
	file := &volumes.File{
		EnvID:     envID,
		Name:      "test.txt",
		Size:      int64(len(content)),
		Path:      s.tmpdir,
		IsPublic:  false,
		CreatedAt: utils.Unix{Valid: true, Time: t},
	}

	s.NoError(os.WriteFile(file.FullPath(), content, 0664))
	s.NoError(volumes.Store().Insert(ctx, []*volumes.File{file}, envID))

	return user.JWT(jwt.MapClaims{
		"token":  strings.Replace(authToken, "Bearer ", "", 1),
		"appId":  appID.String(),
		"envId":  envID.String(),
		"fileId": file.ID.String(),
	})
}

func (s *HandlerVolumesDownloadSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	token, err := s.prepareFile(usertest.Authorization(usr.ID), app.ID, env.ID)
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/volumes/download?token=%s", token),
		nil,
		nil,
	)

	headers := response.Header()

	s.Equal(http.StatusOK, response.Code)
	s.Equal(headers.Get("Content-Length"), "12")
	s.Equal(headers.Get("Content-Type"), "text/plain; charset=utf-8")
	s.Equal(headers.Get("Content-Disposition"), "attachment; filename=test.txt")
	s.Equal(headers.Get("Last-Modified"), "Sat, 06 Apr 2024 15:45:30 GMT")
	s.Equal("Hello world!", response.String())
}

func (s *HandlerVolumesDownloadSuite) Test_Fail_InvalidHash() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/volumes/download?token=invalid-hash",
		nil,
		nil,
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerVolumesDownloadSuite) Test_Fail_MissingHash() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/volumes/download",
		nil,
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerVolumesDownload(t *testing.T) {
	suite.Run(t, &HandlerVolumesDownloadSuite{})
}
