package factory

import (
	"context"
	"encoding/json"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type MockEnv struct {
	*buildconf.Env
	*Factory
}

func (e MockEnv) Insert(conn databasetest.TestDB) error {
	var mailer any
	var data any
	var err error

	if e.Data != nil {
		data, err = json.Marshal(e.Data)

		if err != nil {
			return err
		}
	}

	if e.MailerConf != nil {
		mailer, err = e.MailerConf.Bytes()

		if err != nil {
			return err
		}
	}

	params := []any{
		e.AppID, e.Name, e.Branch, data, e.AutoPublish,
		e.AutoDeployBranches, e.AutoDeploy, e.DeletedAt, mailer,
		e.SchemaConf, e.AuthConf,
	}

	return conn.PrepareOrPanic(`
		INSERT INTO apps_build_conf (
			app_id, env_name, branch, build_conf, auto_publish,
			auto_deploy_branches, auto_deploy, deleted_at, mailer_conf,
			schema_conf, auth_conf
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING env_id;`,
	).QueryRow(params...).Scan(&e.ID)
}

// GetEnv returns the first environment that was created
// in this factory. If none is found, it creates a new one.
func (f *Factory) GetEnv() *MockEnv {
	if res := factoryLookup[MockEnv](f); res != nil {
		return res
	}

	return f.MockEnv(nil)
}

func (f *Factory) MockEnv(app *MockApp, overwrites ...map[string]any) *MockEnv {
	if app == nil {
		app = f.GetApp()
	}

	env := &buildconf.Env{
		AppID:       app.ID,
		Name:        "production",
		Branch:      "main",
		AutoPublish: false,
		AutoDeploy:  true,
		DeletedAt:   utils.Unix{},
		Data: &buildconf.BuildConf{
			BuildCmd:   "npm run build",
			DistFolder: "build",
			Vars: map[string]string{
				"NODE_ENV": "production",
			},
		},
	}

	for _, o := range overwrites {
		merge(env, o)
	}

	mock := f.newObject(MockEnv{
		Env:     env,
		Factory: f,
	}).(MockEnv)

	if err := mock.Insert(f.conn); err != nil {
		panic(err)
	}

	return &mock
}

type MockAPIKey struct {
	*apikey.Token
	*Factory
}

func (k MockAPIKey) Insert(conn databasetest.TestDB) error {
	return apikey.NewStore().AddAPIKey(context.Background(), k.Token)
}

func (f *Factory) MockAPIKey(app *MockApp, env *MockEnv, overwrites ...map[string]any) *MockAPIKey {
	if app == nil {
		app = f.GetApp()
	}

	if env == nil {
		env = f.GetEnv()
	}

	key := &apikey.Token{
		AppID: app.ID,
		EnvID: env.ID,
		Name:  "Default",
		Scope: apikey.SCOPE_ENV,
		Value: apikey.GenerateTokenValue(),
	}

	for _, o := range overwrites {
		merge(key, o)
	}

	mock := f.newObject(MockAPIKey{
		Token:   key,
		Factory: f,
	}).(MockAPIKey)

	if err := mock.Insert(f.conn); err != nil {
		panic(err)
	}

	return &mock

}
