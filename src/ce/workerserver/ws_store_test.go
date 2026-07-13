package jobs_test

import (
	"context"
	"testing"

	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type WSStoreSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *WSStoreSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *WSStoreSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *WSStoreSuite) TestLogsDeletingDeployments() {
	app := s.MockApp(nil)
	timeStamp := int64(1657134279)
	mock := s.MockAppLog(app, map[string]any{
		"Timestamp": timeStamp,
	})

	store := jobs.NewStore()

	err := store.RemoveOldLogs(context.Background())

	a := assert.New(s.T())
	a.Equal(mock.Timestamp, timeStamp)
	a.NoError(err)
}

func TestWSStore(t *testing.T) {
	suite.Run(t, &WSStoreSuite{})
}
