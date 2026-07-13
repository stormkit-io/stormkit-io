package applog_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/applog"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/suite"
)

type LogStoreSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *LogStoreSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *LogStoreSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *LogStoreSuite) Test_InsertLog() {
	mockApp := s.Factory.MockApp(nil, nil)
	mockEnv := s.Factory.MockEnv(mockApp, nil)
	mockDeployment := s.Factory.MockDeployment(mockEnv)

	logs := []*applog.Log{{
		AppID:         mockApp.App.ID,
		HostName:      "1",
		Timestamp:     0,
		DeploymentID:  mockDeployment.ID,
		EnvironmentID: mockEnv.ID,
		RequestID:     "1234",
		Label:         "test",
		Data:          "data1",
	},
		{
			AppID:         mockApp.App.ID,
			HostName:      "2",
			Timestamp:     0,
			RequestID:     "45",
			DeploymentID:  mockDeployment.ID,
			EnvironmentID: mockEnv.ID,
			Label:         "test_2",
			Data:          "data2",
		}}

	err := applog.NewStore().InsertLogs(context.Background(), logs)
	s.NoError(err)

	var count int
	err = s.conn.QueryRow("SELECT COUNT(*) FROM app_logs").Scan(&count)
	s.NoError(err)
	s.Equal(2, count)
}

func TestStore(t *testing.T) {
	suite.Run(t, &LogStoreSuite{})
}
