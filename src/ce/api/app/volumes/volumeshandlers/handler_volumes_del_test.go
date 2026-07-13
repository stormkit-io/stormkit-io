package volumeshandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
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
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerVolumesDelSuite struct {
	suite.Suite
	*factory.Factory
	conn            databasetest.TestDB
	tmpdir          string
	appID           types.ID
	envID           types.ID
	usrID           types.ID
	files           []*volumes.File
	originalEncrypt utils.EncryptToStringFunc
}

func (s *HandlerVolumesDelSuite) SetupSuite() {
	tmpDir, err := os.MkdirTemp("", "tmp-volumes-")
	s.originalEncrypt = utils.EncryptToString
	s.tmpdir = tmpDir
	s.NoError(err)
}

func (s *HandlerVolumesDelSuite) BeforeTest(suiteName, _ string) {
	utils.EncryptToString = func(plaintext string, altKey ...[]byte) string {
		return "random-token"
	}

	utils.NewUnix = factory.MockNewUnix

	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	t, err := time.Parse(time.DateTime, "2024-04-06 15:45:30")
	s.NoError(err)

	cfg := admin.InstanceConfig{
		VolumesConfig: &admin.VolumesConfig{
			MountType: volumes.FileSys,
			RootPath:  s.tmpdir,
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cfg))

	files := []*volumes.File{
		{Name: "text.js", Path: s.tmpdir, IsPublic: false, CreatedAt: utils.Unix{Valid: true, Time: t}},
		{Name: "folder/index.js", Path: path.Join(s.tmpdir, "folder"), IsPublic: true, CreatedAt: utils.Unix{Valid: true, Time: t}},
	}

	s.NoError(os.MkdirAll(path.Join(s.tmpdir, "folder"), 0775))
	s.NoError(os.WriteFile(path.Join(files[0].Path, files[0].Name), []byte("Hello world"), 0664))
	s.NoError(os.WriteFile(path.Join(files[1].Path, path.Base(files[1].Name)), []byte("Hello world"), 0664))
	s.NoError(volumes.Store().Insert(context.Background(), files, env.ID))

	s.envID = env.ID
	s.appID = app.ID
	s.usrID = usr.ID
	s.files = files

	admin.SetMockLicense()
}

func (s *HandlerVolumesDelSuite) AfterTest(_, _ string) {
	utils.EncryptToString = s.originalEncrypt
	utils.NewUnix = factory.OriginalNewUnix
	admin.ResetMockLicense()
	s.conn.CloseTx()
}

func (s *HandlerVolumesDelSuite) TearDownSuite() {
	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}
}

func (s *HandlerVolumesDelSuite) Test_Success() {
	info, err := os.Stat(path.Join(s.files[0].Path, s.files[0].Name))
	s.NoError(err)
	s.NotNil(info)

	ids := strings.Join([]string{
		s.files[0].ID.String(),
		s.files[1].ID.String(),
	}, ",")

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/volumes?appId=%s&envId=%s&ids=%s", s.appID.String(), s.envID.String(), ids),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.usrID),
		},
	)

	expected := fmt.Sprintf(`{
		"removed": [
			{ "id": "%s", "createdAt": 1712418330, "isPublic": true, "name": "folder/index.js", "size": 0, "publicLink": "%s", "mountType": "filesys" },
			{ "id": "%s", "createdAt": 1712418330, "isPublic": false, "name": "text.js", "size": 0, "mountType": "filesys" }
		]
	}`,
		s.files[1].ID.String(),
		s.files[1].PublicLink(),
		s.files[0].ID.String(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())

	info, err = os.Stat(path.Join(s.files[0].Path, s.files[0].Name))
	s.True(os.IsNotExist(err))
	s.Nil(info)

	files, err := volumes.Store().SelectFiles(context.Background(), volumes.SelectFilesArgs{
		FileID: []types.ID{s.files[0].ID, s.files[1].ID},
	})

	s.NoError(err)
	s.Len(files, 0)

}

func TestHandlerVolumesDel(t *testing.T) {
	suite.Run(t, &HandlerVolumesDelSuite{})
}
