package factory

import (
	"context"
	"fmt"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

var appCounter = 0

type MockApp struct {
	*app.App
	*Factory
}

func (a MockApp) Insert(conn databasetest.TestDB) error {
	if !a.CreatedAt.Valid {
		a.CreatedAt = utils.NewUnix()
		a.CreatedAt.Time = time.Unix(1700489144, 0).UTC()
	}

	repo := null.NewString(a.Repo, a.Repo != "")

	return conn.PrepareOrPanic(`
		INSERT INTO apps (
			private_key, client_secret, repo, display_name,
			user_id, client_id, auto_deploy, artifacts_deleted,
			deploy_trigger, runtime, created_at, team_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING app_id;`,
	).QueryRow(
		a.PrivateKey(context.Background()), a.ClientSecret, &repo, a.DisplayName,
		a.UserID, a.ClientID, a.AutoDeploy, false,
		null.NewString(a.DeployTrigger, a.DeployTrigger != ""),
		a.Runtime, a.CreatedAt, a.TeamID,
	).Scan(&a.ID)
}

// GetApp returns the first application that was created
// in this factory. If none is found, it will create a new one.
func (f *Factory) GetApp() *MockApp {
	if res := factoryLookup[MockApp](f); res != nil {
		return res
	}

	return f.MockApp(nil)
}

func (f *Factory) MockApp(usr *MockUser, overwrites ...map[string]any) *MockApp {
	if usr == nil {
		usr = f.GetUser()
	}

	appCounter = appCounter + 1
	appl := app.New(usr.ID)
	appl.TeamID = usr.DefaultTeamID
	appl.Repo = "github/svedova/react-minimal"
	appl.DisplayName = fmt.Sprintf("react-minimal-%d", appCounter)

	for _, o := range overwrites {
		merge(appl, o)
	}

	mock := f.newObject(MockApp{
		App:     appl,
		Factory: f,
	}).(MockApp)

	err := mock.Insert(f.conn)

	if err != nil {
		panic(err)
	}

	return &mock
}
