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

type HandlerVolumesSizeSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerVolumesSizeSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerVolumesSizeSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerVolumesSizeSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	ctx := context.Background()

	t, err := time.Parse(time.DateTime, "2024-04-06 15:45:30")
	s.NoError(err)

	err = volumes.Store().Insert(ctx, []*volumes.File{
		{Name: "test.txt", Size: 13, Path: "", IsPublic: false, CreatedAt: utils.Unix{Valid: true, Time: t}},
		{Name: "public.txt", Size: 10, Path: "", IsPublic: true, CreatedAt: utils.Unix{Valid: true, Time: t}},
	}, env.ID)

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(volumeshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/volumes/size?appId=%s&envId=%s", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{ "size": 23 }`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerVolumesSize(t *testing.T) {
	suite.Run(t, &HandlerVolumesSizeSuite{})
}
