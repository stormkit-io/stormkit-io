package volumeshandlers_test

import (
	"context"
	"net/http"
	"os"
	"strings"
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

type HandlerVolumesPostSuite struct {
	suite.Suite
	*factory.Factory
	conn         databasetest.TestDB
	tmpdir       string
	originalFunc func() utils.Unix
}

func (s *HandlerVolumesPostSuite) SetupSuite() {
	tmpDir, err := os.MkdirTemp("", "tmp-volumes-")
	s.tmpdir = tmpDir
	s.NoError(err)
}

func (s *HandlerVolumesPostSuite) BeforeTest(suiteName, _ string) {
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
}

func (s *HandlerVolumesPostSuite) AfterTest(_, _ string) {
	utils.NewUnix = s.originalFunc
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerVolumesPostSuite) TearDownSuite() {
	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}
}

func (s *HandlerVolumesPostSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	ctx := context.Background()
	cfg := admin.InstanceConfig{
		VolumesConfig: &admin.VolumesConfig{
			MountType: volumes.FileSys,
			RootPath:  s.tmpdir,
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cfg))

	requestBody, contentType, err := shttptest.MultipartForm(map[string][]byte{
		"appId": []byte(app.ID.String()),
		"envId": []byte(env.ID.String()),
	}, map[string][]shttptest.UploadFile{
		"files": {
			{Name: "test.txt", Data: "Hello world!\n"},
			{Name: "some:file.txt", Data: "Hi World!\n"},
			{Name: "test/file.txt", Data: "How are you?\n"},
			{Name: "test/file.txt", Data: "Hey there?\n"}, // Duplicate - should take precedence
		},
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/volumes",
		requestBody,
		map[string]string{
			"Content-Type":  contentType,
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"failed": {},
		"files": [
			{ "createdAt": 1712418330, "name": "test.txt", "size": 13, "isPublic": false, "id": "1", "mountType": "filesys" },
			{ "createdAt": 1712418330, "name": "some:file.txt", "size": 10, "isPublic": false, "id": "2", "mountType": "filesys" },
			{ "createdAt": 1712418330, "name": "test/file.txt", "size": 11, "isPublic": false, "id": "3", "mountType": "filesys" }
		]
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())

	files, err := volumes.Store().SelectFiles(ctx, volumes.SelectFilesArgs{EnvID: env.ID})
	s.NoError(err)
	s.NotEmpty(files)
	s.Len(files, 3)
	s.Equal("test/file.txt", files[0].Name)
	s.Equal("some:file.txt", files[1].Name)
	s.Equal("test.txt", files[2].Name)

	// Test duplicate insertion (should overwrite existing file)
	requestBody, contentType, err = shttptest.MultipartForm(map[string][]byte{
		"appId": []byte(app.ID.String()),
		"envId": []byte(env.ID.String()),
	}, map[string][]shttptest.UploadFile{
		"files": {
			{Name: "test.txt", Data: "Changed content!\n"},
		},
	})

	s.NoError(err)

	response = shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/volumes",
		requestBody,
		map[string]string{
			"Content-Type":  contentType,
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected = `{
		"failed": {},
		"files": [
			{ "createdAt": 1712418330, "name": "test.txt", "size": 17, "isPublic": false, "id": "1", "mountType": "filesys" }
		]
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())

	files, err = volumes.Store().SelectFiles(ctx, volumes.SelectFilesArgs{EnvID: env.ID})
	s.NoError(err)
	s.NotEmpty(files)
	s.Len(files, 3)
	s.Equal("test/file.txt", files[0].Name)
	s.Equal("some:file.txt", files[1].Name)
	s.Equal("test.txt", files[2].Name)
	s.Equal(int64(1712418330), files[2].UpdatedAt.Unix())
}

func (s *HandlerVolumesPostSuite) Test_NotConfigured() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	requestBody, contentType, err := shttptest.MultipartForm(map[string][]byte{
		"appId": []byte(app.ID.String()),
		"envId": []byte(env.ID.String()),
	}, map[string][]shttptest.UploadFile{
		"files": {
			{Name: "test.txt", Data: "Hello world!\n"},
		},
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/volumes",
		requestBody,
		map[string]string{
			"Content-Type":  contentType,
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{ "error": "Volumes is not yet configured." }`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerVolumesPost(t *testing.T) {
	suite.Run(t, &HandlerVolumesPostSuite{})
}
