package deployhandlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhooks"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerDeployCallbackSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
	mockClient       *mocks.ClientInterface
}

func (s *HandlerDeployCallbackSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	deployhooks.StatusChecksEnabled = false
	s.mockCacheService = &mocks.CacheInterface{}
	s.mockClient = &mocks.ClientInterface{}
	appcache.DefaultCacheService = s.mockCacheService
	integrations.CachedClient = s.mockClient
	config.SetIsStormkitCloud(true)
}

func (s *HandlerDeployCallbackSuite) AfterTest(_, _ string) {
	deployhooks.StatusChecksEnabled = true
	s.mockCacheService = nil
	appcache.DefaultCacheService = nil
	integrations.CachedClient = nil
	s.conn.CloseTx()
}

func (s *HandlerDeployCallbackSuite) Test_CommitInfo_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployment(env, map[string]any{
		"Commit": deploy.CommitInfo{},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId": utils.EncryptID(depl.ID),
			"commit": map[string]string{
				"sha":     "6acde42",
				"author":  "David Lorenzo",
				"message": "fix: user navigation",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(1),
		},
	)

	d, err := deploy.NewStore().DeploymentByID(context.Background(), depl.ID)
	s.NoError(err)
	s.Equal(http.StatusOK, response.Code)
	s.Equal("6acde42", d.Commit.ID.ValueOrZero())
	s.Equal("David Lorenzo", d.Commit.Author.ValueOrZero())
	s.Equal("fix: user navigation", d.Commit.Message.ValueOrZero())
}

func (s *HandlerDeployCallbackSuite) Test_GithubRunID_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployment(env)
	runID := "581785151"

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId": utils.EncryptID(depl.ID),
			"runId":    runID,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	d, err := deploy.NewStore().DeploymentByID(context.Background(), depl.ID)
	s.NoError(err)
	s.Equal(http.StatusOK, response.Code)
	s.Equal(runID, utils.Int64ToString(d.GithubRunID.ValueOrZero()))
}

func (s *HandlerDeployCallbackSuite) Test_Logs_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployment(env)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId": utils.EncryptID(depl.ID),
			"logs":     "Hello world",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	d, err := deploy.
		NewStore().
		MyDeployment(context.Background(), &deploy.DeploymentsQueryFilters{
			DeploymentID: depl.ID,
			IncludeLogs:  utils.Ptr(true),
		})

	s.NoError(err)
	s.Equal(http.StatusOK, response.Code)
	s.Equal("Hello world", d.Logs.ValueOrZero())
}

func (s *HandlerDeployCallbackSuite) Test_StatusChecksLogs_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployment(env, map[string]any{
		"ExitCode": null.IntFrom(0),
	})

	res := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId": utils.EncryptID(depl.ID),
			"logs":     "my-logs",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	ds, err := deploy.NewStore().MyDeployments(context.Background(), &deploy.DeploymentsQueryFilters{
		DeploymentID: depl.ID,
		IncludeLogs:  utils.Ptr(true),
	})

	s.Len(ds, 1)

	s.NoError(err)
	s.Equal(http.StatusOK, res.Code)
	s.Equal("my-logs", ds[0].StatusChecks.ValueOrZero())
}

func (s *HandlerDeployCallbackSuite) Test_ExitCode_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployment(env, map[string]any{
		"ShouldPublish": true,
	})

	manifest := &deploy.BuildManifest{
		CDNFiles: []deploy.CDNFile{
			{Name: "index.html", Headers: map[string]string{"etag": "12345"}},
		},
	}

	uploadResult := integrations.UploadResult{
		Client: integrations.UploadOverview{
			FilesUploaded: 15,
			BytesUploaded: 5000,
			Location:      "local:/path/to/deployment/client",
		},
	}

	// Should publish the deployment
	s.mockCacheService.On("Reset", env.ID).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId":        utils.EncryptID(depl.ID),
			"manifest":        manifest,
			"result":          uploadResult,
			"outcome":         "success",
			"hasStatusChecks": false,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	d, err := deploy.NewStore().DeploymentByID(context.Background(), depl.ID)

	s.NoError(err)
	s.Equal(int64(0), d.ExitCode.ValueOrZero())
	s.Equal(manifest, d.BuildManifest)
	s.Equal(uploadResult.Client.Location, d.UploadResult.ClientLocation)
	s.Equal(uploadResult.Client.BytesUploaded, d.UploadResult.ClientBytes)
	s.Empty(d.UploadResult.ServerBytes)
	s.Empty(d.UploadResult.ServerLocation)
	s.Empty(d.UploadResult.ServerlessBytes)
	s.Empty(d.UploadResult.ServerlessLocation)
	s.Empty(d.UploadResult.MigrationsBytes)
	s.Empty(d.UploadResult.MigrationsLocation)

	depls, err := deploy.NewStore().MyDeployments(context.Background(), &deploy.DeploymentsQueryFilters{
		DeploymentID: d.ID,
		Published:    utils.Ptr(true),
	})

	s.NoError(err)
	s.Len(depls, 1)
}

func (s *HandlerDeployCallbackSuite) Test_ExitCode_Success_With_StatusChecks() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployment(env, map[string]any{
		"ShouldPublish": true,
	})

	manifest := &deploy.BuildManifest{
		CDNFiles: []deploy.CDNFile{
			{Name: "index.html", Headers: map[string]string{"etag": "12345"}},
		},
	}

	uploadResult := integrations.UploadResult{
		Client: integrations.UploadOverview{
			FilesUploaded: 15,
			BytesUploaded: 5000,
			Location:      "local:/path/to/deployment/client",
		},
	}

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId":        utils.EncryptID(depl.ID),
			"manifest":        manifest,
			"result":          uploadResult,
			"outcome":         "success",
			"hasStatusChecks": true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	// This tests simply that we do not publish
	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerDeployCallbackSuite) Test_ExitCode_EmptyManifest() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployments(2, env, map[string]any{
		"CreatedAt": utils.UnixFrom(time.Now().Add(-8 * time.Minute)),
	})

	manifest := &deploy.BuildManifest{
		FunctionHandler: "server.js:handler",
	}

	uploadResult := integrations.UploadResult{
		Client: integrations.UploadOverview{
			FilesUploaded: 15,
			BytesUploaded: 5000,
			Location:      "local:/path/to/deployment/client",
		},
		Server: integrations.UploadOverview{
			BytesUploaded: 500,
			Location:      "local:/path/to/deployment/server",
		},
		API: integrations.UploadOverview{
			BytesUploaded: 2500,
			Location:      "local:/path/to/deployment/api",
		},
	}

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId": utils.EncryptID(depl[0].ID),
			"manifest": manifest,
			"outcome":  "success",
			"result":   uploadResult,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	d, err := deploy.NewStore().DeploymentByID(context.Background(), depl[0].ID)

	s.NoError(err)
	s.Equal(int64(0), d.ExitCode.ValueOrZero())
	s.Equal(manifest, d.BuildManifest)
	s.Equal(uploadResult.Client.Location, d.UploadResult.ClientLocation)
	s.Equal(uploadResult.Client.BytesUploaded, d.UploadResult.ClientBytes)
	s.Equal(uploadResult.Server.Location, d.UploadResult.ServerLocation)
	s.Equal(uploadResult.Server.BytesUploaded, d.UploadResult.ServerBytes)
	s.Equal(uploadResult.API.Location, d.UploadResult.ServerlessLocation)
	s.Equal(uploadResult.API.BytesUploaded, d.UploadResult.ServerlessBytes)
	s.True(d.ExitCode.Valid)

	metrics, err := user.NewStore().UserMetrics(context.Background(), user.UserMetricsArgs{UserID: usr.ID})
	s.NoError(err)
	s.NotNil(metrics)
	s.Equal(int64(0), metrics.FunctionInvocations)
	s.Equal(int64(8000), metrics.StorageUsedInBytes) // 5000 + 500 + 2500
	s.Equal(int64(0), metrics.BandwidthUsedInBytes)
	s.Equal(int64(8), metrics.BuildMinutes)
}

func (s *HandlerDeployCallbackSuite) Test_ExitCode_InstallRuntimes() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployments(2, env)

	mockMise := &mocks.MiseInterface{}
	mise.DefaultMise = mockMise
	config.SetIsSelfHosted(true)

	defer func() {
		mise.DefaultMise = nil
		config.SetIsSelfHosted(false)
	}()

	mockMise.On("InstallMise", mock.Anything).Return(nil).Once()
	mockMise.On("InstallGlobal", mock.Anything, "go@1.24").Return("", nil).Once()
	mockMise.On("InstallGlobal", mock.Anything, "node@22").Return("", nil).Once()
	mockMise.On("Prune", mock.Anything).Return(nil).Once()

	manifest := &deploy.BuildManifest{
		Runtimes: []string{"go@1.24", "node@22"},
	}

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId": utils.EncryptID(depl[0].ID),
			"manifest": manifest,
			"outcome":  "success",
			"result":   integrations.UploadResult{},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerDeployCallbackSuite) Test_LockDeployment_StatusChecksPassed() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployment(env, map[string]any{
		"ExitCode": null.IntFrom(0),
	})

	// Should publish the deployment
	s.mockCacheService.On("Reset", env.ID).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId":        utils.EncryptID(depl.ID),
			"outcome":         "success",
			"hasStatusChecks": true,
			"lock":            true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	ds, err := deploy.NewStore().MyDeployments(context.Background(), &deploy.DeploymentsQueryFilters{
		DeploymentID: depl.ID,
	})

	s.NoError(err)
	s.Len(ds, 1)
	s.Equal(int64(0), ds[0].ExitCode.ValueOrZero())
	s.True(ds[0].ExitCode.Valid)
	s.True(ds[0].IsLocked())
	s.True(ds[0].StatusChecksPassed.ValueOrZero())
}

func (s *HandlerDeployCallbackSuite) Test_LockDeployment_StatusChecksNotPassed() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployment(env, map[string]any{
		"ExitCode": null.IntFrom(0),
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId":        utils.EncryptID(depl.ID),
			"outcome":         "failed",
			"hasStatusChecks": true,
			"lock":            true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	ds, err := deploy.NewStore().MyDeployments(context.Background(), &deploy.DeploymentsQueryFilters{
		DeploymentID: depl.ID,
	})

	s.NoError(err)
	s.Len(ds, 1)
	s.Equal(int64(0), ds[0].ExitCode.ValueOrZero())
	s.True(ds[0].ExitCode.Valid)
	s.True(ds[0].IsLocked())
	s.False(ds[0].StatusChecksPassed.ValueOrZero())
}

func (s *HandlerDeployCallbackSuite) Test_ShouldNotOverwrite() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	depl := s.MockDeployment(env, map[string]any{
		"IsImmutable": null.BoolFrom(true),
		"ExitCode":    null.IntFrom(1),
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId": utils.EncryptID(depl.ID),
			"outcome":  "success",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusConflict, response.Code)

	d, err := deploy.NewStore().DeploymentByID(context.Background(), depl.ID)
	s.NoError(err)
	s.Equal(int64(1), d.ExitCode.ValueOrZero())
	s.Equal(true, d.ExitCode.Valid)
}

func (s *HandlerDeployCallbackSuite) Test_InvalidDeployID() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/callback",
		map[string]any{
			"deployId": "1",
			"commit": map[string]string{
				"sha":     "6acde42",
				"author":  "David Lorenzo",
				"message": "fix: user navigation",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerDeployCallbackSuite) Test_RunMigrations() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			Host:              s.conn.Cfg.Host,
			Port:              s.conn.Cfg.Port,
			DBName:            s.conn.Cfg.DBName,
			SchemaName:        s.conn.Cfg.Schema,
			MigrationPassword: s.conn.Cfg.Password,
			MigrationUserName: s.conn.Cfg.User,
			MigrationsEnabled: true,
		},
	})

	migrationsFile := "local:/path/to/migrations.zip"

	files := map[string][]byte{
		"002_create_users.sql": []byte("CREATE TABLE test_users (id INT);"),
		"003_add_index.sql":    []byte("CREATE INDEX idx_test_users ON test_users(id);"),
	}

	zipContent, err := file.ZipInMemory(files)
	s.NoError(err)

	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location: migrationsFile,
	}).Return(&integrations.GetFileResult{
		Content: zipContent,
	}, nil).Once()

	s.NoError(deployhandlers.RunMigrations(context.Background(), env.ID, migrationsFile))

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeMigrations)
	s.NoError(err)

	migrations, err := store.Migrations(context.Background())
	s.NoError(err)

	s.Len(migrations, 2)
	s.Equal("002_create_users.sql", migrations[0].Name)
	s.Equal("003_add_index.sql", migrations[1].Name)
}

func TestCallbackHandler(t *testing.T) {
	suite.Run(t, &HandlerDeployCallbackSuite{})
}
