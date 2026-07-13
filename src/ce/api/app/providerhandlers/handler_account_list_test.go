package providerhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/providerhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HandlerAccountListSuite struct {
	suite.Suite
	*factory.Factory
	conn     databasetest.TestDB
	ghClient *mocks.GithubClient
}

func (s *HandlerAccountListSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.ghClient = &mocks.GithubClient{}
	github.DefaultGithubClient = s.ghClient
}

func (s *HandlerAccountListSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	github.DefaultGithubClient = nil
}

func (s *HandlerAccountListSuite) Test_RepoList_GitHub() {
	s.ghClient.On("ListUserInstallations", mock.Anything, &github.ListOptions{PerPage: 100}).Return([]*github.Installation{
		{
			ID: utils.Ptr(int64(1)),
			Account: &github.User{
				ID:        utils.Ptr(int64(1)),
				Login:     utils.Ptr("stormkit"),
				AvatarURL: utils.Ptr("https://stormkit.io/avatar.png"),
			},
		},
	}, nil, nil).Once()

	usr := s.MockUser(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(providerhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/provider/github/accounts",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"accounts": [{
			"id": "1",
			"login": "stormkit",
			"avatar": "https://stormkit.io/avatar.png"
		}]
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerAccountList(t *testing.T) {
	suite.Run(t, new(HandlerAccountListSuite))
}
