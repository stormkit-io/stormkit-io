package volumeshandlers_test

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes/volumeshandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerVolumesFileSuite struct {
	suite.Suite
	*factory.Factory
	conn   databasetest.TestDB
	tmpdir string
}

func (s *HandlerVolumesFileSuite) SetupSuite() {
	tmpDir, err := os.MkdirTemp("", "tmp-volumes-")
	s.tmpdir = tmpDir
	s.NoError(err)
}

func (s *HandlerVolumesFileSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerVolumesFileSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerVolumesFileSuite) TearDownSuite() {
	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}
}

func (s *HandlerVolumesFileSuite) prepareFile(isPublic bool, envID types.ID) string {
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
		IsPublic:  isPublic,
		CreatedAt: utils.Unix{Valid: true, Time: t},
	}

	s.NoError(os.WriteFile(file.FullPath(), content, 0664))
	s.NoError(volumes.Store().Insert(ctx, []*volumes.File{file}, envID))

	return file.PublicLink()
}

func (s *HandlerVolumesFileSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	parsed, err := url.Parse(s.prepareFile(true, env.ID))
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		parsed.Path,
		nil,
		nil,
	)

	s.Equal(http.StatusOK, response.Code)
	s.Equal(response.Header().Get("Content-Length"), "12")
	s.Equal(response.Header().Get("Content-Type"), "text/plain; charset=utf-8")
	s.Equal(response.Header().Get("Last-Modified"), "Sat, 06 Apr 2024 15:45:30 GMT")
	s.Equal("Hello world!", response.String())
}

func (s *HandlerVolumesFileSuite) Test_Fail_NotPublic() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	parsed, err := url.Parse(s.prepareFile(false, env.ID))
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		parsed.Path,
		nil,
		nil,
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerVolumesFileSuite) Test_Fail_InvalidHash() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/volumes/file/invalid-hash",
		nil,
		nil,
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerVolumesFile(t *testing.T) {
	suite.Run(t, &HandlerVolumesFileSuite{})
}
