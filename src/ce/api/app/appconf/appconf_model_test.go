package appconf_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type appconfSuite struct {
	suite.Suite
	*factory.Factory

	manifest *deploy.BuildManifest
	depls    []*factory.MockDeployment
	depl     *factory.MockDeployment
	user     *factory.MockUser
	app      *factory.MockApp
	env      *factory.MockEnv
	ctx      context.Context
	domains  []*buildconf.DomainModel
	conn     databasetest.TestDB
}

func (s *appconfSuite) SetupSuite() {
	s.conn = databasetest.InitTx("appconf_model_suite")
	s.Factory = factory.New(s.conn)

	s.ctx = context.Background()
	s.user = s.MockUser()
	s.app = s.MockApp(s.user)
	s.env = s.MockEnv(s.app, map[string]any{
		"Data": &buildconf.BuildConf{
			BuildCmd:   "npm run build",
			ServerCmd:  "node index.js",
			ErrorFile:  "/index.html",
			DistFolder: "build",
			Vars: map[string]string{
				"NODE_ENV": "production",
			},
			Redirects: []redirects.Redirect{
				{From: "/path-1", To: "/path-2"},
				{From: "/path-3", To: "/path-4"},
			},
		},
	})

	s.domains = []*buildconf.DomainModel{
		{
			AppID:      s.app.ID,
			EnvID:      s.env.ID,
			Name:       "example.org",
			Verified:   true,
			VerifiedAt: utils.NewUnix(),
		},
		{
			AppID:      s.app.ID,
			EnvID:      s.env.ID,
			Name:       "www.example.org",
			Verified:   true,
			VerifiedAt: utils.NewUnix(),
		},
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), s.domains[0]))
	s.NoError(buildconf.DomainStore().Insert(context.Background(), s.domains[1]))

	s.manifest = &deploy.BuildManifest{
		Redirects: []deploy.Redirect{
			{From: "/my-path", To: "/my-new-path", Status: http.StatusPermanentRedirect},
		},
	}

	s.depl = s.MockDeployment(s.env, map[string]any{
		"UploadResult": &deploy.UploadResult{
			ClientLocation: "aws:s3-bucket-name/s3-key-prefix",
			ServerLocation: "aws:arn:aws:lambda:eu-central-1:account-id:function:lambda-name",
		},
		"BuildManifest": s.manifest,
	})

	// Create 3 additional deployments to increase test reliability
	s.depls = s.MockDeployments(3, s.env)

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "S1", Enabled: true, Prepend: false, Location: "body", AppID: s.app.ID, EnvID: s.env.ID},
		{Title: "Snippet 2", Content: "S2", Enabled: true, Prepend: true, Location: "body", AppID: s.app.ID, EnvID: s.env.ID},
		{Title: "Snippet 3", Content: "S3", Enabled: false, Prepend: false, Location: "head", AppID: s.app.ID, EnvID: s.env.ID},
		{Title: "Snippet 4", Content: "S4", Enabled: true, Prepend: true, Location: "head", AppID: s.app.ID, EnvID: s.env.ID},
		{Title: "Snippet 5", Content: "S5", Enabled: true, Prepend: true, Location: "head", AppID: s.app.ID, EnvID: s.env.ID, Rules: &buildconf.SnippetRule{Hosts: []string{"example.org"}}},
	}

	s.NoError(buildconf.SnippetsStore().Insert(s.ctx, snippets))

	// Publish deployments
	settings := []*deploy.PublishSettings{
		{
			EnvID:        s.env.ID,
			DeploymentID: s.depl.ID,
			Percentage:   25,
			NoCacheReset: true,
		},
		{
			EnvID:        s.env.ID,
			DeploymentID: s.depls[1].ID,
			Percentage:   75,
			NoCacheReset: true,
		},
	}

	s.NoError(deploy.Publish(context.Background(), settings))
}

func (s *appconfSuite) AfterTest(_, _ string) {
	authwall.Store().SetAuthWallConfig(context.Background(), s.env.ID, nil)
}

func (s *appconfSuite) TearDownSuite() {
	s.conn.CloseTx()
	shttp.DefaultRequest = nil
}

func (s *appconfSuite) Test_ByDeploymentID() {
	s.NoError(authwall.Store().SetAuthWallConfig(context.Background(), s.env.ID, &authwall.Config{
		Status: "all",
	}))

	configs, err := appconf.NewStore().Configs(s.ctx, appconf.ConfigFilters{
		DeploymentID: s.depl.ID,
		DisplayName:  s.app.DisplayName,
	})

	s.NoError(err)
	s.Len(configs, 1)
	s.Equal(s.depl.ID, configs[0].DeploymentID)
	s.Equal(s.depl.AppID, configs[0].AppID)
	s.Equal(float64(25), configs[0].Percentage)
	s.Equal(types.ID(0), configs[0].DomainID)
	s.Equal("all", configs[0].AuthWall)
	s.Equal("node index.js", configs[0].ServerCmd)
	s.Equal("aws:arn:aws:lambda:eu-central-1:account-id:function:lambda-name", configs[0].FunctionLocation)
	s.Equal("aws:s3-bucket-name/s3-key-prefix", configs[0].StorageLocation)
	s.Equal(&appconf.SnippetInjection{
		HeadPrepend: "S4",
		BodyAppend:  "S1",
		BodyPrepend: "S2",
	}, appconf.SnippetsHTML(configs[0].Snippets))
}

func (s *appconfSuite) Test_ByDeploymentID_Snippets_ProdDomain() {
	s.NoError(authwall.Store().SetAuthWallConfig(context.Background(), s.env.ID, &authwall.Config{
		Status: "dev",
	}))

	configs, err := appconf.NewStore().Configs(s.ctx, appconf.ConfigFilters{
		HostName: "example.org",
	})

	s.NoError(err)
	s.Len(configs, 2)
	s.Equal("", configs[0].CertKey)
	s.Equal("", configs[0].CertValue)
	s.Equal(s.depl.ID, configs[0].DeploymentID)
	s.Equal(s.depl.AppID, configs[0].AppID)
	s.Equal(float64(25), configs[0].Percentage)
	s.Equal("dev", configs[0].AuthWall)
	s.Equal("aws:arn:aws:lambda:eu-central-1:account-id:function:lambda-name", configs[0].FunctionLocation)
	s.Equal("aws:s3-bucket-name/s3-key-prefix", configs[0].StorageLocation)
	s.Equal(&appconf.SnippetInjection{
		HeadPrepend: "S4S5",
		BodyAppend:  "S1",
		BodyPrepend: "S2",
	}, appconf.SnippetsHTML(configs[0].Snippets))
}

func (s *appconfSuite) Test_ByDeploymentID_Snippets_DevDomain() {
	snippet := &buildconf.Snippet{
		Title:    "Snippet only for dev domains",
		Content:  "SX",
		Enabled:  true,
		Prepend:  false,
		Location: "body",
		AppID:    s.app.ID,
		EnvID:    s.env.ID,
		Rules: &buildconf.SnippetRule{
			Hosts: []string{"*.dev"},
		},
	}

	s.NoError(buildconf.SnippetsStore().Insert(s.ctx, []*buildconf.Snippet{snippet}))

	defer func() {
		s.NoError(buildconf.SnippetsStore().Delete(s.ctx, []types.ID{snippet.ID}, s.env.ID))
	}()

	configs, err := appconf.NewStore().Configs(s.ctx, appconf.ConfigFilters{
		HostName:     fmt.Sprintf("%s.stormkit:8888", s.app.DisplayName),
		DeploymentID: s.depl.ID,
		DisplayName:  s.app.DisplayName,
	})

	s.NoError(err)
	s.Len(configs, 1)
	s.Equal(s.depl.ID, configs[0].DeploymentID)
	s.Equal(s.depl.AppID, configs[0].AppID)
	s.Equal(types.ID(0), configs[0].DomainID)
	s.Equal(float64(25), configs[0].Percentage)
	s.Equal("", configs[0].AuthWall)
	s.Equal("/index.html", configs[0].ErrorFile)
	s.Equal("aws:arn:aws:lambda:eu-central-1:account-id:function:lambda-name", configs[0].FunctionLocation)
	s.Equal("aws:s3-bucket-name/s3-key-prefix", configs[0].StorageLocation)
	s.Equal(&appconf.SnippetInjection{
		HeadPrepend: "S4",
		BodyAppend:  "S1SX",
		BodyPrepend: "S2",
	}, appconf.SnippetsHTML(configs[0].Snippets))
}

func (s *appconfSuite) Test_ByDomainName() {
	s.NoError(buildconf.DomainStore().UpdateDomainCert(context.Background(), &buildconf.DomainModel{
		ID: s.domains[0].ID,
		CustomCert: &buildconf.CustomCert{
			Value: "cert-value",
			Key:   "cert-key",
		},
	}))

	defer func() {
		s.NoError(buildconf.DomainStore().UpdateDomainCert(context.Background(), &buildconf.DomainModel{
			ID: s.domains[0].ID,
		}))
	}()

	configs, err := appconf.NewStore().Configs(s.ctx, appconf.ConfigFilters{
		HostName: "example.org",
	})

	s.NoError(err)
	s.Len(configs, 2)

	// Deployment 1
	s.Equal(s.depl.ID, configs[0].DeploymentID)
	s.Equal(s.depl.AppID, configs[0].AppID)
	s.Equal(float64(25), configs[0].Percentage)
	s.Equal("cert-value", configs[0].CertValue)
	s.Equal("cert-key", configs[0].CertKey)
	s.Equal(s.domains[0].ID, configs[0].DomainID)
	s.Greater(configs[0].DomainID, types.ID(0))
	s.Equal("aws:arn:aws:lambda:eu-central-1:account-id:function:lambda-name", configs[0].FunctionLocation)
	s.Equal("aws:s3-bucket-name/s3-key-prefix", configs[0].StorageLocation)
	s.Equal(&appconf.SnippetInjection{
		HeadPrepend: "S4S5",
		BodyAppend:  "S1",
		BodyPrepend: "S2",
	}, appconf.SnippetsHTML(configs[0].Snippets))

	// Deployment 2
	s.Equal(s.depls[1].ID, configs[1].DeploymentID)
	s.Equal(s.depls[1].AppID, configs[1].AppID)
	s.Equal(float64(75), configs[1].Percentage)
	s.Equal("", configs[1].FunctionLocation)
	s.Equal("", configs[1].StorageLocation)
	s.Equal(appconf.StaticFileConfig{
		"/about": {FileName: "about", Headers: map[string]string{"accept-encoding": "None", "content-type": "text/html; charset=utf-8"}},
		"/index": {FileName: "index", Headers: map[string]string{"keep-alive": "30", "content-type": "text/html; charset=utf-8"}},
	}, configs[1].StaticFiles)

	s.Equal([]redirects.Redirect{
		// This one is the redirects defined from the UI (takes precedence)
		{From: "/path-1", To: "/path-2"},
		{From: "/path-3", To: "/path-4"},
		// This one is coming from deployment redirects.json
		{From: "/my-path", To: "/my-new-path", Status: 308},
	}, configs[0].Redirects)

	s.Equal(&appconf.SnippetInjection{
		HeadPrepend: "S4S5",
		BodyAppend:  "S1",
		BodyPrepend: "S2",
	}, appconf.SnippetsHTML(configs[0].Snippets))
}

func (s *appconfSuite) Test_ByDisplayName() {
	configs, err := appconf.NewStore().Configs(s.ctx, appconf.ConfigFilters{
		HostName:    fmt.Sprintf("%s--%s.stormkit:8888", s.app.DisplayName, s.env.Name),
		DisplayName: s.app.DisplayName,
		EnvName:     s.env.Name,
	})

	s.NoError(err)
	s.Len(configs, 2)

	// Deployment 1
	s.Equal(s.depl.ID, configs[0].DeploymentID)
	s.Equal(s.depl.AppID, configs[0].AppID)
	s.Equal(float64(25), configs[0].Percentage)
	s.Equal("aws:arn:aws:lambda:eu-central-1:account-id:function:lambda-name", configs[0].FunctionLocation)
	s.Equal("aws:s3-bucket-name/s3-key-prefix", configs[0].StorageLocation)
	s.Equal(&appconf.SnippetInjection{
		HeadPrepend: "S4",
		BodyAppend:  "S1",
		BodyPrepend: "S2",
	}, appconf.SnippetsHTML(configs[0].Snippets))

	// Deployment 2
	s.Equal(s.depls[1].ID, configs[1].DeploymentID)
	s.Equal(s.depls[1].AppID, configs[1].AppID)
	s.Equal(float64(75), configs[1].Percentage)
	s.Equal("", configs[1].FunctionLocation)
	s.Equal("", configs[1].StorageLocation)
	s.Equal(appconf.StaticFileConfig{
		"/about": {FileName: "about", Headers: map[string]string{"accept-encoding": "None", "content-type": "text/html; charset=utf-8"}},
		"/index": {FileName: "index", Headers: map[string]string{"keep-alive": "30", "content-type": "text/html; charset=utf-8"}},
	}, configs[1].StaticFiles)
	s.Equal(&appconf.SnippetInjection{
		HeadPrepend: "S4",
		BodyAppend:  "S1",
		BodyPrepend: "S2",
	}, appconf.SnippetsHTML(configs[0].Snippets))
}

// This test case is useful especially for self-hosted clients where they'd like to
// test additional domains by adding a custom domain.
func (s *appconfSuite) Test_ByStormkitDevSubdomain() {
	domain := &buildconf.DomainModel{
		AppID:      s.app.ID,
		EnvID:      s.env.ID,
		Name:       "my-test.stormkit:8888",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	defer func() {
		s.NoError(buildconf.DomainStore().DeleteDomain(context.Background(), buildconf.DeleteDomainArgs{
			DomainID: domain.ID,
		}))
	}()

	configs, err := appconf.NewStore().Configs(s.ctx, appconf.ConfigFilters{
		HostName:    "my-test.stormkit:8888",
		DisplayName: "my-test",
		EnvName:     "production",
	})

	s.NoError(err)
	s.Len(configs, 2)

	// Deployment 1
	s.Equal(s.depl.ID, configs[0].DeploymentID)
	s.Equal(s.depl.AppID, configs[0].AppID)
	s.Equal(float64(25), configs[0].Percentage)
	s.Equal("aws:arn:aws:lambda:eu-central-1:account-id:function:lambda-name", configs[0].FunctionLocation)
	s.Equal("aws:s3-bucket-name/s3-key-prefix", configs[0].StorageLocation)
	s.Equal(&appconf.SnippetInjection{
		HeadPrepend: "S4",
		BodyAppend:  "S1",
		BodyPrepend: "S2",
	}, appconf.SnippetsHTML(configs[0].Snippets))

	// Deployment 2
	s.Equal(s.depls[1].ID, configs[1].DeploymentID)
	s.Equal(s.depls[1].AppID, configs[1].AppID)
	s.Equal(float64(75), configs[1].Percentage)
	s.Equal("", configs[1].FunctionLocation)
	s.Equal("", configs[1].StorageLocation)
	s.Equal(appconf.StaticFileConfig{
		"/about": {FileName: "about", Headers: map[string]string{"accept-encoding": "None", "content-type": "text/html; charset=utf-8"}},
		"/index": {FileName: "index", Headers: map[string]string{"keep-alive": "30", "content-type": "text/html; charset=utf-8"}},
	}, configs[1].StaticFiles)

	s.Equal(&appconf.SnippetInjection{
		HeadPrepend: "S4",
		BodyAppend:  "S1",
		BodyPrepend: "S2",
	}, appconf.SnippetsHTML(configs[0].Snippets))
}

func TestAppConf(t *testing.T) {
	suite.Run(t, &appconfSuite{})
}
