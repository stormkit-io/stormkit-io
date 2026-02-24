package appconf_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type ShttpSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *ShttpSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *ShttpSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *ShttpSuite) Test_IsStormkitDev() {
	admin.MustConfig().DomainConfig.Dev = "http://stormkit:8888"

	s.True(appconf.IsStormkitDev("my-app--1868181.stormkit:8888"))
	s.True(appconf.IsStormkitDev("my-app.stormkit:8888"))
	s.False(appconf.IsStormkitDev("my-app.stormkit"))
	s.False(appconf.IsStormkitDev("my-app.com"))
	s.False(appconf.IsStormkitDev("stormkit:8888"))

	admin.MustConfig().DomainConfig.Dev = "http://stormkit.dev"

	s.True(appconf.IsStormkitDev("my-app--1868181.stormkit.dev"))
	s.True(appconf.IsStormkitDev("my-app.stormkit.dev"))
	s.False(appconf.IsStormkitDev("my-app.stormkit.dev.io"))
	s.False(appconf.IsStormkitDev("my-app.com"))
	s.False(appconf.IsStormkitDev("stormkit:8888"))
}

func (s *ShttpSuite) Test_ParseHost() {
	admin.MustConfig().DomainConfig.Dev = "http://stormkit.dev"

	customDomains := []string{
		"dev.app",
		"dev.app.stormkit",
		"stormkit.de",
		"my-app--1.my.domain",
	}

	for _, d := range customDomains {
		s.Equal(appconf.RequestContext{
			DomainName:   d,
			DisplayName:  "",
			EnvName:      "",
			DeploymentID: 0,
			App:          nil,
			User:         nil,
		}, *appconf.ParseHost(d))
	}

	devDomains := []appconf.RequestContext{
		{DomainName: "app.stormkit.dev", DisplayName: "app"},
		{DomainName: "dev.app.stormkit.dev", DisplayName: "dev.app"},
		{DomainName: "app--dev.stormkit.dev", DisplayName: "app", EnvName: "dev"},
		{DomainName: "app--1.stormkit.dev", DisplayName: "app", EnvName: "", DeploymentID: types.ID(1)},
	}

	for _, d := range devDomains {
		s.Equal(appconf.RequestContext{
			DomainName:   d.DomainName,
			DisplayName:  d.DisplayName,
			EnvName:      d.EnvName,
			DeploymentID: d.DeploymentID,
			App:          nil,
			User:         nil,
		}, *appconf.ParseHost(d.DomainName))
	}
}

func (s *ShttpSuite) Test_IsStormkitDevStrict() {
	admin.MustConfig().DomainConfig.Dev = "http://stormkit:8888"
	s.False(appconf.IsStormkitDevStrict("my-app--1868181.stormkit:8888"))
	s.True(appconf.IsStormkitDevStrict("stormkit:8888"))
}

func (s *ShttpSuite) Test_FetchAppConf_ByDisplayName() {
	admin.MustConfig().DomainConfig.Dev = "http://stormkit:8888"

	usr := s.MockUser()
	apl := s.MockApp(usr, map[string]any{"DisplayName": "my-app"})
	env := s.MockEnv(apl, map[string]any{
		"Data": &buildconf.BuildConf{
			Vars: map[string]string{
				"ENV_VAR": "value",
			},
		},
	})

	// Let's insert a deployment first
	deployment := &deploy.Deployment{
		AppID:        apl.ID,
		EnvID:        env.ID,
		Branch:       "main",
		ConfigCopy:   []byte(""), // Let's say the config copy is empty, since we want to test it's able to unmarshal the request
		IsAutoDeploy: false,
	}

	deploy.NewStore().InsertDeployment(context.Background(), deployment)

	confs, err := appconf.FetchConfig(fmt.Sprintf("my-app--%d.stormkit:8888", deployment.ID))
	s.NoError(err)
	s.NotNil(confs)
	s.Len(confs, 1)

	conf := confs[0]

	s.Equal(apl.ID, conf.AppID)
	s.Equal(env.ID, conf.EnvID)
	s.Equal(deployment.ID, conf.DeploymentID)
	s.Equal(map[string]string{
		"ENV_VAR":           "value",
		"SK_APP_ID":         apl.ID.String(),
		"SK_ENV_ID":         env.ID.String(),
		"SK_ENV":            "production",
		"SK_DEPLOYMENT_ID":  deployment.ID.String(),
		"SK_DEPLOYMENT_URL": fmt.Sprintf("http://%s--%s.stormkit:8888", apl.DisplayName, deployment.ID),
		"SK_ENV_URL":        fmt.Sprintf("http://%s.stormkit:8888", apl.DisplayName),
		"STORMKIT":          "true",
	}, conf.EnvVariables)

}

func TestShttpMethods(t *testing.T) {
	suite.Run(t, &ShttpSuite{})
}
