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
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerVolumesDownloadURLSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerVolumesDownloadURLSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerVolumesDownloadURLSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerVolumesDownloadURLSuite) prepareFile(envID types.ID) *volumes.File {
	ctx := context.Background()

	t, err := time.Parse(time.DateTime, "2024-04-06 15:45:30")
	s.NoError(err)

	content := []byte("Hello world!")
	file := &volumes.File{
		EnvID:     envID,
		Name:      "test.txt",
		Size:      int64(len(content)),
		Path:      "",
		IsPublic:  false,
		CreatedAt: utils.Unix{Valid: true, Time: t},
	}

	s.NoError(volumes.Store().Insert(ctx, []*volumes.File{file}, envID))

	return file
}

func (s *HandlerVolumesDownloadURLSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	file := s.prepareFile(env.ID)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/volumes/download/url?appId=%d&envId=%d&fileId=%d", app.ID, env.ID, file.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.Contains(response.String(), "http://api.stormkit:8888/volumes/download?token=ey")
}

func TestHandlerVolumesDownloadURL(t *testing.T) {
	suite.Run(t, &HandlerVolumesDownloadURLSuite{})
}
