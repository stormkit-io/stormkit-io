package providerhandlers_test

import (
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
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

type HandlerRepoListSuite struct {
	suite.Suite
	*factory.Factory
	conn     databasetest.TestDB
	ghClient *mocks.GithubClient
}

func (s *HandlerRepoListSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.ghClient = &mocks.GithubClient{}
	github.DefaultGithubClient = s.ghClient
}

func (s *HandlerRepoListSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	github.DefaultGithubClient = nil
}

func (s *HandlerRepoListSuite) Test_RepoList_GitHub() {
	s.ghClient.On("ListRepos", mock.Anything, &github.ListOptions{Page: 1, PerPage: 10}).Return(&github.ListRepositories{
		TotalCount: aws.Int(2),
		Repositories: []*github.Repository{
			{
				ID:       utils.Ptr(int64(1)),
				Name:     utils.Ptr("repo1"),
				FullName: utils.Ptr("stormkit/repo1"),
			},
			{
				ID:       utils.Ptr(int64(2)),
				Name:     utils.Ptr("repo2"),
				FullName: utils.Ptr("stormkit/repo2"),
			},
		},
	}, nil, nil).Once()

	usr := s.MockUser(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(providerhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/provider/github/repos",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"hasNextPage": false,
		"repos": [{
			"fullName": "stormkit/repo1",
			"name": "repo1"
		},
		{
			"fullName": "stormkit/repo2",
			"name": "repo2"
		}]
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerRepoListSuite) Test_RepoList_GitHub_WithSearch() {
	s.ghClient.On("ListRepos", mock.Anything, &github.ListOptions{Page: 1, PerPage: 100}).Return(&github.ListRepositories{
		TotalCount: aws.Int(2),
		Repositories: []*github.Repository{
			{
				ID:       utils.Ptr(int64(1)),
				Name:     utils.Ptr("repo1"),
				FullName: utils.Ptr("stormkit/repo1"),
			},
			{
				ID:       utils.Ptr(int64(2)),
				Name:     utils.Ptr("repo2"),
				FullName: utils.Ptr("stormkit/repo2"),
			},
		},
	}, nil, nil).Once()

	usr := s.MockUser(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(providerhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/provider/github/repos?search=repo1",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"hasNextPage": false,
		"repos": [{
			"fullName": "stormkit/repo1",
			"name": "repo1"
		}]
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerRepoList(t *testing.T) {
	suite.Run(t, new(HandlerRepoListSuite))
}
