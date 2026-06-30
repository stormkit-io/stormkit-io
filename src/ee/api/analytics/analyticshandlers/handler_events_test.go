package analyticshandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics/analyticshandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerEventsSuite struct {
	suite.Suite
	*factory.Factory

	conn     databasetest.TestDB
	user     *factory.MockUser
	env      *factory.MockEnv
	domainID types.ID
}

func (s *HandlerEventsSuite) SetupSuite() {
	s.conn = databasetest.InitTx("handler_events_suite")
	s.Factory = factory.New(s.conn)

	admin.SetMockLicense()

	s.user = s.MockUser()
	appl := s.MockApp(s.user)
	s.env = s.MockEnv(appl)
	domain := &buildconf.DomainModel{
		AppID:      appl.ID,
		EnvID:      s.env.ID,
		Name:       "example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))
	s.domainID = domain.ID

	now := utils.UnixFrom(time.Now())

	event := func(name, visitor, ref string) analytics.Event {
		return analytics.Event{
			AppID:       appl.ID,
			EnvID:       s.env.ID,
			DomainID:    domain.ID,
			VisitorID:   null.NewString(visitor, visitor != ""),
			EventName:   name,
			RequestPath: null.StringFrom("/"),
			EventTS:     now,
			Metadata:    null.StringFrom(`{"ref": "` + ref + `"}`),
		}
	}

	events := []analytics.Event{
		event("trip_creation", "v1", "mobile"),
		event("trip_creation", "v2", "web"),
		event("trip_creation", "v1", "mobile"), // repeat visitor
	}

	s.NoError(analytics.NewStore().InsertEvents(context.Background(), events))
	s.NoError(jobs.SyncAnalyticsEvents(context.Background()))
}

func (s *HandlerEventsSuite) TearDownSuite() {
	admin.ResetMockLicense()
	s.conn.CloseTx()
}

func (s *HandlerEventsSuite) request(path string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(analyticshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		path,
		nil,
		map[string]string{"Authorization": usertest.Authorization(s.user.ID)},
	)
}

func (s *HandlerEventsSuite) Test_Events_Success() {
	res := s.request(fmt.Sprintf("/analytics/events?envId=%s&domainId=%d", s.env.ID, s.domainID))

	s.Equal(http.StatusOK, res.Code)
	s.JSONEq(`[{"name":"trip_creation","total":3,"unique":2}]`, res.String())
}

func (s *HandlerEventsSuite) Test_Properties_Success() {
	res := s.request(fmt.Sprintf("/analytics/events/properties?envId=%s&domainId=%d&event=trip_creation", s.env.ID, s.domainID))

	s.Equal(http.StatusOK, res.Code)
	s.JSONEq(`["ref"]`, res.String())
}

func (s *HandlerEventsSuite) Test_Breakdown_Success() {
	res := s.request(fmt.Sprintf("/analytics/events/breakdown?envId=%s&domainId=%d&event=trip_creation&property=ref", s.env.ID, s.domainID))

	s.Equal(http.StatusOK, res.Code)
	s.JSONEq(`[{"name":"mobile","total":2,"unique":1},{"name":"web","total":1,"unique":1}]`, res.String())
}

func (s *HandlerEventsSuite) Test_MissingDomainID_BadRequest() {
	res := s.request(fmt.Sprintf("/analytics/events?envId=%s", s.env.ID))

	s.Equal(http.StatusBadRequest, res.Code)
}

func (s *HandlerEventsSuite) Test_DomainFromAnotherEnv_Forbidden() {
	otherUser := s.MockUser()
	otherApp := s.MockApp(otherUser)
	otherEnv := s.MockEnv(otherApp)
	otherDomain := &buildconf.DomainModel{
		AppID:      otherApp.ID,
		EnvID:      otherEnv.ID,
		Name:       "tenant-b.example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), otherDomain))

	res := s.request(fmt.Sprintf("/analytics/events?envId=%s&domainId=%d", s.env.ID, otherDomain.ID))

	s.Equal(http.StatusForbidden, res.Code)
}

func TestHandlerEvents(t *testing.T) {
	suite.Run(t, &HandlerEventsSuite{})
}
