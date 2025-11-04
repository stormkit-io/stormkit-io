package hosting_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type BatcherSuite struct {
	suite.Suite

	originalQueueName string
	originalMaxItems  int
}

func (s *BatcherSuite) SetupSuite() {
	hosting.MaxItems = 1
	hosting.QueueName = "batch_analytics_test"
	s.originalQueueName = hosting.QueueName
	s.originalMaxItems = hosting.MaxItems
}

func (s *BatcherSuite) TearDownSuite() {
	rediscache.Client().Del(context.Background(), hosting.QueueName)
	hosting.QueueName = s.originalQueueName
	hosting.MaxItems = s.originalMaxItems
	hosting.Batcher = nil
}

func (s *BatcherSuite) Test_Queueing() {
	record := &jobs.HostingRecord{
		AppID:         types.ID(5),
		EnvID:         types.ID(10),
		DeploymentID:  types.ID(15),
		BillingUserID: types.ID(20),
		HostName:      "www.stormkit.io",
		Analytics: &analytics.Record{
			RequestPath: "/test-path",
			RequestTS:   utils.NewUnix(),
		},
	}

	s.NoError(hosting.Queue(record))

	s.Eventually(func() bool {
		msg, err := rediscache.Client().LPop(context.Background(), hosting.QueueName).Result()
		readRecord := jobs.HostingRecord{}

		s.NoError(err)
		s.NoError(json.Unmarshal([]byte(msg), &readRecord))

		// Normalize for comparison
		readRecord.Analytics.RequestTS = utils.UnixFrom(record.Analytics.RequestTS.Time)

		s.Equal(*record, readRecord)

		return true
	}, 5*time.Second, 100*time.Millisecond, "Expected 1 item in the queue")

}

func TestBatcherSuite(t *testing.T) {
	suite.Run(t, &BatcherSuite{})
}
