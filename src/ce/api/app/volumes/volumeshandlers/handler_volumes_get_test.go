package volumeshandlers_test

import (
	"context"
	"fmt"
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

type HandlerVolumesGetSuite struct {
	suite.Suite
	*factory.Factory
	conn            databasetest.TestDB
	originalEncrypt utils.EncryptToStringFunc
}

func (s *HandlerVolumesGetSuite) SetupSuite() {
	s.originalEncrypt = utils.EncryptToString
}

func (s *HandlerVolumesGetSuite) BeforeTest(suiteName, _ string) {
	utils.EncryptToString = func(plaintext string, altKey ...[]byte) string {
		return "random-token"
	}

	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	admin.SetMockLicense()
}

func (s *HandlerVolumesGetSuite) AfterTest(_, _ string) {
	utils.EncryptToString = s.originalEncrypt
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerVolumesGetSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	ctx := context.Background()

	t, err := time.Parse(time.DateTime, "2024-04-06 15:45:30")
	s.NoError(err)

	files := []*volumes.File{
		{Name: "test.txt", Size: 13, Path: "", IsPublic: false, CreatedAt: utils.Unix{Valid: true, Time: t}},
		{Name: "public.txt", Size: 10, Path: "", IsPublic: true, CreatedAt: utils.Unix{Valid: true, Time: t}},
	}

	err = volumes.Store().Insert(ctx, files, env.ID)

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/volumes?appId=%s&envId=%s", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"files": [
			{ "createdAt": 1712418330, "name": "public.txt", "size": 10, "isPublic": true, "id": "%s", "publicLink": "%s", "mountType": "filesys" },
			{ "createdAt": 1712418330, "name": "test.txt", "size": 13, "isPublic": false, "id": "%s", "mountType": "filesys" }
		]
	}`,
		files[1].ID.String(),
		files[1].PublicLink(),
		files[0].ID.String(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerVolumesGet(t *testing.T) {
	suite.Run(t, &HandlerVolumesGetSuite{})
}
