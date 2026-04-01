package jobs_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type JobHandlerForwardTest struct {
	suite.Suite
	*factory.Factory
	conn   databasetest.TestDB
	ctx    context.Context
	client *rediscache.RedisCache
}

func (s *JobHandlerForwardTest) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.ctx = context.Background()
	s.client = rediscache.Client()

	// Clear the queue before each test
	s.client.Del(s.ctx, jobs.HostingQueueName)
}

func (s *JobHandlerForwardTest) AfterTest(suiteName, _ string) {
	// Clean up queue after each test
	s.client.Del(s.ctx, jobs.HostingQueueName)
	s.conn.CloseTx()
}

func (s *JobHandlerForwardTest) createTestRecord() jobs.HostingRecord {
	return jobs.HostingRecord{
		AppID:         types.ID(123),
		EnvID:         types.ID(456),
		DeploymentID:  types.ID(789),
		BillingUserID: types.ID(321),
		HostName:      "example.com",
		Logs: []integrations.Log{
			{
				Timestamp: time.Now().Unix(),
				Level:     "info",
				Message:   "Test log message",
			},
		},
		Analytics: &analytics.Record{
			AppID:       types.ID(123),
			EnvID:       types.ID(456),
			VisitorIP:   "192.168.1.1",
			HostName:    "example.com",
			UserAgent:   null.StringFrom("Mozilla/5.0"),
			RequestPath: "/test",
			RequestTS:   utils.UnixFrom(time.Now()),
			StatusCode:  200,
		},
		TotalBandwidth:  1024,
		FunctionInvoked: true,
	}
}

func (s *JobHandlerForwardTest) pushToQueue(record jobs.HostingRecord) {
	data, err := json.Marshal(record)
	s.NoError(err)
	s.client.RPush(s.ctx, jobs.HostingQueueName, string(data))
}

func (s *JobHandlerForwardTest) Test_IngestHandlerForward_EmptyQueue() {
	// Test with empty queue
	err := jobs.IngestHandlerForward(s.ctx)
	s.NoError(err)

	// Verify queue is still empty
	length := s.client.LLen(s.ctx, jobs.HostingQueueName).Val()
	s.Equal(int64(0), length)
}

func (s *JobHandlerForwardTest) Test_IngestHandlerForward_MultipleRecords() {
	// Create and push multiple test records
	records := []jobs.HostingRecord{
		s.createTestRecord(),
		{
			AppID:           types.ID(456),
			EnvID:           types.ID(789),
			DeploymentID:    types.ID(12),
			BillingUserID:   types.ID(654),
			HostName:        "test.com",
			Logs:            []integrations.Log{},
			Analytics:       nil, // No analytics for this record
			TotalBandwidth:  2048,
			FunctionInvoked: false,
		},
	}

	for _, record := range records {
		s.pushToQueue(record)
	}

	// Process the queue
	err := jobs.IngestHandlerForward(s.ctx)
	s.NoError(err)

	// Verify queue is empty after processing
	length := s.client.LLen(s.ctx, jobs.HostingQueueName).Val()
	s.Equal(int64(0), length)
}

func (s *JobHandlerForwardTest) Test_IngestHandlerForward_BatchLimit() {
	s.T().Setenv("STORMKIT_HOSTING_QUEUE_BATCH_SIZE", "100")

	// Push more than 100 records to test batch limit
	for i := 0; i < 150; i++ {
		record := s.createTestRecord()
		record.AppID = types.ID(i)
		s.pushToQueue(record)
	}

	// Verify we have 150 records in queue
	length := s.client.LLen(s.ctx, jobs.HostingQueueName).Val()
	s.Equal(int64(150), length)

	// Process the queue (should only process 100)
	err := jobs.IngestHandlerForward(s.ctx)
	s.NoError(err)

	// Verify 50 records remain in queue
	remainingLength := s.client.LLen(s.ctx, jobs.HostingQueueName).Val()
	s.Equal(int64(50), remainingLength)
}

func (s *JobHandlerForwardTest) Test_IngestHandlerForward_RecordWithoutAnalytics() {
	// Create record without analytics
	record := s.createTestRecord()
	record.Analytics = nil
	s.pushToQueue(record)

	// Process the queue
	err := jobs.IngestHandlerForward(s.ctx)
	s.NoError(err)

	// Verify queue is empty
	length := s.client.LLen(s.ctx, jobs.HostingQueueName).Val()
	s.Equal(int64(0), length)
}

func (s *JobHandlerForwardTest) Test_IngestHandlerForward_RecordWithoutLogs() {

	// Create record without logs
	record := s.createTestRecord()
	record.Logs = []integrations.Log{}
	s.pushToQueue(record)

	// Process the queue
	err := jobs.IngestHandlerForward(s.ctx)
	s.NoError(err)

	// Verify queue is empty
	length := s.client.LLen(s.ctx, jobs.HostingQueueName).Val()
	s.Equal(int64(0), length)
}

func TestJobHandlerForwardTest(t *testing.T) {
	suite.Run(t, &JobHandlerForwardTest{})
}
