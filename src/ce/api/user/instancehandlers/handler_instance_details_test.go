package instancehandlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/go-github/v71/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/instancehandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HandlerVersionSuite struct {
	suite.Suite
	*factory.Factory

	conn       databasetest.TestDB
	mockClient mocks.ReleaseClient
	original   config.VersionConfig
}

func (s *HandlerVersionSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Github: admin.GithubConfig{
			Account: "my-stormkit-app",
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.TODO(), cnf))

	conf := config.Get()
	s.original = conf.Version
	s.mockClient = mocks.ReleaseClient{}
	instancehandlers.GHClient = &s.mockClient
	conf.Version = config.VersionConfig{
		Tag:  "v1.7.30",
		Hash: "d390d39d542bed4464216223fc22851f7a7e73d3",
	}

	admin.SetMockLicense()
}

func (s *HandlerVersionSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.Get().Version = s.original
	instancehandlers.GHClient = nil
	instancehandlers.LastCacheTime = time.Time{}
	instancehandlers.LatestRelease = ""
	instancehandlers.LatestCommit = ""
	admin.ResetMockLicense()
}

func (s *HandlerVersionSuite) Test_Success_SelfHosted() {
	config.SetIsSelfHosted(true)

	s.mockClient.On("GetLatestRelease", mock.Anything, "stormkit-io", "bin").Return(&github.RepositoryRelease{
		TagName: aws.String("v1.8.25"),
	}, nil, nil).Once()

	response := shttptest.Request(
		shttp.NewRouter().RegisterService(instancehandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/instance",
		nil,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(response.String(), `{ 
		"stormkit": {
			"apiCommit": "d390d39",
			"apiVersion": "v1.7.30",
			"edition": "self-hosted"
		},
		"latest": {
			"apiVersion": "v1.8.25"
		},
		"license": { 
			"seats": 10,
			"remaining": 10,
			"edition": "enterprise"
		},
		"auth": {
			"github": "my-stormkit-app"
		}
	}`)
}

func (s *HandlerVersionSuite) Test_Success_Cloud_NotLoggedIn() {
	config.SetIsStormkitCloud(true)

	s.mockClient.On("GetLatestRelease", mock.Anything, "stormkit-io", "bin").Return(&github.RepositoryRelease{
		TagName: aws.String("v1.8.25"),
	}, nil, nil).Once()

	response := shttptest.Request(
		shttp.NewRouter().RegisterService(instancehandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/instance",
		nil,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ 
		"stormkit": {
			"apiCommit": "d390d39",
			"apiVersion": "v1.7.30",
			"edition": "cloud"
		},
		"latest": {
			"apiVersion": "v1.8.25"
		},
		"auth": {
			"github": "my-stormkit-app"
		}
	}`, response.String())
}

func (s *HandlerVersionSuite) Test_Success_Cloud_LoggedIn() {
	usr := s.MockUser(map[string]any{
		"Metadata": user.UserMeta{
			SeatsPurchased: 3,
			PackageName:    config.PackageUltimate,
		},
	})

	config.SetIsStormkitCloud(true)

	s.mockClient.On("GetLatestRelease", mock.Anything, "stormkit-io", "bin").Return(&github.RepositoryRelease{
		TagName: aws.String("v1.8.25"),
	}, nil, nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(instancehandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/instance",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ 
		"stormkit": {
			"apiCommit": "d390d39",
			"apiVersion": "v1.7.30",
			"edition": "cloud"
		},
		"latest": {
			"apiVersion": "v1.8.25"
		},
		"license": {
			"seats": 3,
			"remaining": 2,
			"edition": "enterprise"
		},
		"auth": {
			"github": "my-stormkit-app"
		}
	}`, response.String())
}

func (s *HandlerVersionSuite) Test_ShouldCacheGithubResponse() {
	s.mockClient.On("GetLatestRelease", mock.Anything, "stormkit-io", "bin").Return(&github.RepositoryRelease{
		TagName: aws.String("v1.8.25"),
	}, nil, nil).Once()

	for i := 0; i < 3; i = i + 1 {
		response := shttptest.Request(
			shttp.NewRouter().RegisterService(instancehandlers.Services).Router().Handler(),
			shttp.MethodGet,
			"/instance",
			nil,
		)

		s.Equal(http.StatusOK, response.Code)
	}
}

func TestHandlerVersionSuite(t *testing.T) {
	suite.Run(t, &HandlerVersionSuite{})
}
