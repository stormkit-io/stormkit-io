package hosting

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/pool"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

var (
	ctx           = context.Background()
	QueueName     = jobs.HostingQueueName
	FlushInterval = time.Second
	MaxItems      = int(1000)
	Batcher       *pool.Buffer
	mu            sync.Mutex
	artifactsWG   sync.WaitGroup
)

func Queue(record *jobs.HostingRecord) error {
	mu.Lock()
	defer mu.Unlock()

	if Batcher == nil {
		Batcher = pool.New(
			pool.WithSize(MaxItems),
			pool.WithFlushInterval(FlushInterval),
			pool.WithFlusher(pool.FlusherFunc(func(items []any) {
				marshaled := []any{}

				for _, log := range items {
					data, err := json.Marshal(log)

					if err != nil {
						slog.Errorf("error while marshaling log: %s", err.Error())
						return
					}

					marshaled = append(marshaled, data)
				}

				if len(marshaled) == 0 {
					return
				}

				pipe := rediscache.Client().Pipeline()

				if _, err := pipe.LPush(ctx, QueueName, marshaled...).Result(); err != nil {
					slog.Errorf("error while pushing into the queue: %v", err)
				}

				if _, err := pipe.Exec(ctx); err != nil {
					slog.Errorf("error while executing the queue: %v", err)
				}
			})),
		)
	}

	return Batcher.Push(record)
}
