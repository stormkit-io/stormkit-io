package publicapiv1_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerVolumesPostPublicSuite struct {
	suite.Suite
	*factory.Factory

	conn         databasetest.TestDB
	tmpdir       string
	originalFunc func() utils.Unix
}

func (s *HandlerVolumesPostPublicSuite) SetupSuite() {
	tmpDir, err := os.MkdirTemp("", "tmp-volumes-public-")
	s.NoError(err)
	s.tmpdir = tmpDir
}

func (s *HandlerVolumesPostPublicSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.originalFunc = utils.NewUnix

	utils.NewUnix = func() utils.Unix {
		t, err := time.Parse(time.DateTime, "2024-04-06 15:45:30")
		s.NoError(err)
		return utils.Unix{Valid: true, Time: t}
	}

	admin.ResetCache(context.Background())
	admin.SetMockLicense()
	volumes.CachedFileSys = nil
	volumes.CachedAWS = nil
}

func (s *HandlerVolumesPostPublicSuite) AfterTest(_, _ string) {
	utils.NewUnix = s.originalFunc
	s.conn.CloseTx()
	admin.ResetMockLicense()
	volumes.CachedFileSys = nil
	volumes.CachedAWS = nil
}

func (s *HandlerVolumesPostPublicSuite) TearDownSuite() {
	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}
}

func (s *HandlerVolumesPostPublicSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	key := s.MockAPIKey(app, env)
	ctx := context.Background()
	cfg := admin.InstanceConfig{
		VolumesConfig: &admin.VolumesConfig{
			MountType: volumes.FileSys,
			RootPath:  s.tmpdir,
		},
	}

	s.NoError(admin.Store().UpsertConfig(ctx, cfg))
	admin.ResetCache(ctx)

	requestBody, contentType, err := shttptest.MultipartForm(nil, map[string][]shttptest.UploadFile{
		"files": {
			{Name: "hello.txt", Data: "Hello world!\n"},
			{Name: "assets/logo.png", Data: "PNG DATA"},
			{Name: "assets/logo.png", Data: "PNG DATA v2"}, // Duplicate - last wins
		},
	})
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/volumes",
		requestBody,
		map[string]string{
			"Content-Type":  contentType,
			"Authorization": key.Value,
		},
	)

	expected := `{
		"failed": {},
		"files": [
			{ "createdAt": 1712418330, "id": "1", "isPublic": false, "name": "hello.txt", "size": 13 },
			{ "createdAt": 1712418330, "id": "2", "isPublic": false, "name": "assets/logo.png", "size": 11 }
		]
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())

	files, err := volumes.Store().SelectFiles(ctx, volumes.SelectFilesArgs{EnvID: env.ID})
	s.NoError(err)
	s.Len(files, 2)
}

func (s *HandlerVolumesPostPublicSuite) Test_StorageNotConfigured() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	key := s.MockAPIKey(app, env)

	requestBody, contentType, err := shttptest.MultipartForm(nil, map[string][]shttptest.UploadFile{
		"files": {
			{Name: "hello.txt", Data: "Hello!"},
		},
	})
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/volumes",
		requestBody,
		map[string]string{
			"Content-Type":  contentType,
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error": "File storage is not yet configured."}`, response.String())
}

func (s *HandlerVolumesPostPublicSuite) Test_NoFiles() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	key := s.MockAPIKey(app, env)
	ctx := context.Background()
	cfg := admin.InstanceConfig{
		VolumesConfig: &admin.VolumesConfig{
			MountType: volumes.FileSys,
			RootPath:  s.tmpdir,
		},
	}

	s.NoError(admin.Store().UpsertConfig(ctx, cfg))
	admin.ResetCache(ctx)

	requestBody, contentType, err := shttptest.MultipartForm(nil, nil)
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/volumes",
		requestBody,
		map[string]string{
			"Content-Type":  contentType,
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error": "At least one file is required. Send files under the \"files\" field."}`, response.String())
}

func (s *HandlerVolumesPostPublicSuite) Test_Unauthorized() {
	requestBody, contentType, err := shttptest.MultipartForm(nil, map[string][]shttptest.UploadFile{
		"files": {
			{Name: "hello.txt", Data: "Hello!"},
		},
	})
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/volumes",
		requestBody,
		map[string]string{
			"Content-Type": contentType,
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerVolumesPostPublic(t *testing.T) {
	suite.Run(t, &HandlerVolumesPostPublicSuite{})
}
