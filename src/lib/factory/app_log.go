package factory

import (
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/applog"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
)

type MockAppLog struct {
	*applog.Log
	*Factory
}

func (u MockAppLog) Insert(conn databasetest.TestDB) error {
	conn.PrepareOrPanic(`
    	INSERT INTO
    	skitapi.app_logs (app_id, timestamp, log_data, env_id, deployment_id)
      VALUES
    	($1, $2, random()::text, $3, $4);
	`).QueryRow(u.AppID, u.Timestamp, u.EnvironmentID, u.DeploymentID)

	return nil
}

func (f *Factory) MockAppLog(app *MockApp, overwrites ...map[string]any) *MockAppLog {
	if app == nil {
		app = f.GetApp()
	}

	env := f.MockEnv(app)
	deployment := f.MockDeployment(env)

	log := &applog.Log{
		AppID:         app.ID,
		HostName:      "",
		Timestamp:     time.Now().Unix(),
		RequestID:     "",
		Label:         "",
		Data:          "",
		EnvironmentID: env.ID,
		DeploymentID:  deployment.ID,
	}

	for _, o := range overwrites {
		merge(log, o)
	}

	mockLog := f.newObject(MockAppLog{
		Log:     log,
		Factory: f,
	}).(MockAppLog)

	mockLog.Insert(f.conn)

	return &mockLog
}
