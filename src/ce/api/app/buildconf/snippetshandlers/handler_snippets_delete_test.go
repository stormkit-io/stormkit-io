package snippetshandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerSnippetsDeleteSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
}

func (s *HandlerSnippetsDeleteSuite) SetupSuite() {
	s.mockCacheService = &mocks.CacheInterface{}
}

func (s *HandlerSnippetsDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	appcache.DefaultCacheService = s.mockCacheService
	admin.SetMockLicense()
}

func (s *HandlerSnippetsDeleteSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
	admin.ResetMockLicense()
}

func (s *HandlerSnippetsDeleteSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	ctx := context.Background()

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "<script>console.log('snippet 1')</script>", Enabled: false, Prepend: false, Location: "body", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 2", Content: "<script>console.log('snippet 2')</script>", Enabled: true, Prepend: true, Location: "body", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 3", Content: "<script>console.log('snippet 3')</script>", Enabled: false, Prepend: false, Location: "head", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 4", Content: "<script>console.log('snippet 4')</script>", Enabled: true, Prepend: true, Location: "head", AppID: app.ID, EnvID: env.ID},
	}

	s.NoError(buildconf.SnippetsStore().Insert(ctx, snippets))

	s.mockCacheService.On("Reset", env.ID).Return(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf(
			"/snippets?ids=1,2&envId=%s&appId=%s",
			env.ID.String(),
			app.ID.String(),
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	var err error
	snippets, err = buildconf.SnippetsStore().SnippetsByEnvID(ctx, buildconf.SnippetFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Equal(http.StatusOK, response.Code)
	s.Len(snippets, 2)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "DELETE:SNIPPET",
		EnvName:     env.Name,
		EnvID:       env.ID,
		AppID:       app.ID,
		TeamID:      app.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				Snippets: []string{"1", "2"},
			},
		},
	}, audits[0])
}

func (s *HandlerSnippetsDeleteSuite) TestInvalidRequest_InvalidID() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf(
			"/snippets?ids=1,abc&envId=%s&appId=%s",
			env.ID.String(),
			app.ID.String(),
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"errors": ["ID should be an integer."]}`)
}

func TestHandlerSnippetsDelete(t *testing.T) {
	suite.Run(t, &HandlerSnippetsDeleteSuite{})
}
