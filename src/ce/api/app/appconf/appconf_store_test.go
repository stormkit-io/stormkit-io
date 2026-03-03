package appconf_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type AppConfStoreSuite struct {
	suite.Suite
}

func (s *AppConfStoreSuite) Test_NormalizeHeaders() {
	type HeaderTest struct {
		FileName        string
		ActualHeaders   map[string]string
		ExpectedHeaders map[string]string
	}

	tests := []HeaderTest{
		{
			FileName:      "/subfolder/index.html",
			ActualHeaders: map[string]string{},
			ExpectedHeaders: map[string]string{
				"content-type": "text/html; charset=utf-8",
			},
		},
		{
			FileName:      "/subfolder/index.js",
			ActualHeaders: map[string]string{},
			ExpectedHeaders: map[string]string{
				"content-type": "application/javascript; charset=utf-8",
			},
		},
		{
			FileName:      "/subfolder/index.css",
			ActualHeaders: map[string]string{},
			ExpectedHeaders: map[string]string{
				"content-type": "text/css; charset=utf-8",
			},
		},
		{
			FileName:      "/subfolder/favicon.ico",
			ActualHeaders: map[string]string{},
			ExpectedHeaders: map[string]string{
				"content-type": "image/x-icon",
			},
		},
		{
			FileName: "/subfolder/favicon.png",
			ActualHeaders: map[string]string{
				"content-type": "custom-type",
			},
			ExpectedHeaders: map[string]string{
				"content-type": "custom-type",
			},
		},
	}

	for _, test := range tests {
		file := appconf.StaticFile{
			FileName: test.FileName,
			Headers:  appconf.NormalizeHeaders(test.FileName, test.ActualHeaders),
		}

		// This will be added dynamically
		file.Headers["content-type"] = test.ExpectedHeaders["content-type"]

		s.Equal(file, appconf.StaticFile{
			FileName: test.FileName,
			Headers:  test.ExpectedHeaders,
		})
	}
}

func (s *AppConfStoreSuite) Test_BelongsToEnv() {
	conn := databasetest.InitTx(s.T().Name())
	defer conn.CloseTx()

	factory := factory.New(conn)
	usr := factory.MockUser(nil)
	app := factory.MockApp(usr, nil)
	env := factory.MockEnv(app, nil)

	domainVerified := &buildconf.DomainModel{
		AppID:    app.ID,
		EnvID:    env.ID,
		Name:     "my.example.org",
		Token:    null.NewString("my-token", true),
		Verified: true,
	}

	domainUnverified := &buildconf.DomainModel{
		AppID:    app.ID,
		EnvID:    env.ID,
		Name:     "unverified.example.org",
		Token:    null.NewString("unverified-token", true),
		Verified: false,
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domainUnverified))
	s.NoError(buildconf.DomainStore().Insert(context.Background(), domainVerified))

	store := appconf.NewStore()
	ctx := s.T().Context()

	// Empty domain should not belong to the environment
	belongs, err := store.BelongsToEnv(ctx, env.ID, &appconf.RequestContext{DomainName: ""})
	s.NoError(err)
	s.False(belongs)

	// Verified domain should belong to the environment
	belongs, err = store.BelongsToEnv(ctx, env.ID, &appconf.RequestContext{DomainName: "my.example.org"})
	s.NoError(err)
	s.True(belongs)

	// Unverified domain should belong to the environment
	belongs, err = store.BelongsToEnv(ctx, env.ID, &appconf.RequestContext{DomainName: "unverified.example.org"})
	s.NoError(err)
	s.False(belongs)

	// Inexistent domain should not belong to the environment
	belongs, err = store.BelongsToEnv(ctx, env.ID, &appconf.RequestContext{DomainName: "otherdomain.com"})
	s.NoError(err)
	s.False(belongs)

	// DisplayName + EnvName should belong to the environment
	belongs, err = store.BelongsToEnv(ctx, env.ID, &appconf.RequestContext{DisplayName: app.DisplayName, EnvName: env.Name})
	s.NoError(err)
	s.True(belongs)

	// DisplayName + wrong EnvName should not belong to the environment
	belongs, err = store.BelongsToEnv(ctx, env.ID, &appconf.RequestContext{DisplayName: app.DisplayName, EnvName: "wrong-env"})
	s.NoError(err)
	s.False(belongs)

	// Just DisplayName should belong to the environment
	belongs, err = store.BelongsToEnv(ctx, env.ID, &appconf.RequestContext{DisplayName: app.DisplayName})
	s.NoError(err)
	s.True(belongs)

	// Just wrong DisplayName should not belong to the environment
	belongs, err = store.BelongsToEnv(ctx, env.ID, &appconf.RequestContext{DisplayName: "wrong-display-name"})
	s.NoError(err)
	s.False(belongs)
}

func TestAppConfStoreSui(t *testing.T) {
	suite.Run(t, new(AppConfStoreSuite))
}
