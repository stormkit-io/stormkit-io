package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/ce/api/applog"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var HostingQueueName = "hosting_forward_queue"
var ingestContext = context.Background()

type HostingRecord struct {
	AppID           types.ID           `json:"appId"`
	EnvID           types.ID           `json:"envId"`
	DeploymentID    types.ID           `json:"deploymentId"`
	BillingUserID   types.ID           `json:"billingUserId"`
	HostName        string             `json:"hostName"`
	Logs            []integrations.Log `json:"logs"`
	Analytics       *analytics.Record  `json:"analytics"`
	TotalBandwidth  int64              `json:"totalBandwidth"`
	FunctionInvoked bool               `json:"functionInvoked"`
}

// IngestHandlerForward reads rows from redis and inserts them into the database.
// The batch size defaults to 1000 and can be overridden via STORMKIT_HOSTING_QUEUE_BATCH_SIZE.
//
// Here's an example calculation on finding the number of data we can handle:
// - Let's assume this method is called every 5 seconds
// - Executions per minute:   12
// - Executions per hour:     720
// - Executions per day:      17'280
// - Daily records handled:   17'280 x 1000 = 17'280'000
// - Monthly records handled: 518'400'000
func IngestHandlerForward(ctx context.Context) error {
	client := rediscache.Client()
	analyticsRecords := []analytics.Record{}
	logRecords := []*applog.Log{}
	stats := map[string]map[string]int64{} // userId -> metric -> value
	rows := utils.StringToInt(os.Getenv("STORMKIT_HOSTING_QUEUE_BATCH_SIZE"))

	if rows <= 0 {
		rows = 1000
	}

	msgs, err := client.LPopCount(ctx, HostingQueueName, rows).Result()

	if rediscache.IsConnectionError(err) {
		return err
	}

	if err != nil && !errors.Is(err, redis.Nil) {
		slog.Errorf("error while popping from redis: %v", err)
		return err
	}

	for _, msg := range msgs {
		record := HostingRecord{}

		if err := json.Unmarshal([]byte(msg), &record); err != nil {
			slog.Errorf("cannot unmarshal log: %v", err)
			continue
		}

		if record.Analytics != nil {
			analyticsRecords = append(analyticsRecords, *record.Analytics)
		}

		if len(record.Logs) > 0 {
			for _, log := range record.Logs {
				logRecords = append(logRecords, &applog.Log{
					AppID:         record.AppID,
					DeploymentID:  record.DeploymentID,
					EnvironmentID: record.EnvID,
					HostName:      record.HostName,
					Timestamp:     log.Timestamp,
					Label:         log.Level,
					Data:          log.Message,
				})
			}
		}

		userID := record.BillingUserID.String()

		if stats[userID] == nil {
			stats[userID] = map[string]int64{
				"totalBandwidth":  0,
				"functionInvoked": 0,
			}
		}

		stats[userID]["totalBandwidth"] += record.TotalBandwidth

		if record.FunctionInvoked {
			stats[userID]["functionInvoked"]++
		}
	}

	if len(analyticsRecords) > 0 {
		if err := analytics.NewStore().InsertRecords(analyticsContext, analyticsRecords); err != nil {
			slog.Errorf("error while batch inserting analytic records: %v", err)
		}
	}

	if len(logRecords) > 0 {
		if err := applog.NewStore().InsertLogs(ingestContext, logRecords); err != nil {
			slog.Errorf("error while batch inserting log records: %v", err)
		}
	}

	userStats := []user.Usage{}

	for userID, metrics := range stats {
		userStats = append(userStats, user.Usage{
			UserID:              types.ID(utils.StringToID(userID)),
			BandwidthInBytes:    metrics["totalBandwidth"],
			FunctionInvocations: metrics["functionInvoked"],
		})
	}

	if len(userStats) > 0 && config.IsStormkitCloud() {
		if err := user.NewStore().UpdateUsageMetrics(ingestContext, userStats); err != nil {
			slog.Errorf("error while updating user usage metrics: %v", err)
		}
	}

	return nil
}
