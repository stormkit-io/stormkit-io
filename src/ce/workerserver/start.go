package jobs

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/tasks"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
)

func scheduler() {
	scheduler, err := NewScheduler()

	if err != nil {
		slog.Errorf("cannot start the scheduler: %s", err.Error())
	}

	ctx := context.Background()
	node := NewNode(Options{})

	node.OnStart(func(n *Node) {
		scheduler.RegisterReplicaTasks(ctx)
		slog.Infof("registered replica tasks: %s", n.ID())
	})

	node.OnLeaderElected(func(n *Node) {
		scheduler.RegisterMasterTasks(ctx)
		slog.Infof("registered master tasks: %s", n.ID())
	})

	node.OnLeaderRenounced(func(n *Node) {
		slog.Infof("renounced node as leader: %s", n.ID())
		scheduler.StopMasterTasks()
	})

	node.Start(ctx)

	scheduler.Start()
}

type ServerOpts struct {
	Scheduler bool
}

func Server(serverOpts ...ServerOpts) (*asynq.Server, *asynq.ServeMux) {
	opts := ServerOpts{
		Scheduler: true,
	}

	if len(serverOpts) > 0 {
		opts = serverOpts[0]
	}

	if opts.Scheduler {
		scheduler()
	}

	// TODO: Move tasks.DeploymentStart and TriggerFunctionHttp here
	service := rediscache.Service()
	handlers := map[string]rediscache.Handler{
		rediscache.EventMiseUpdate:           mise.AutoUpdate,
		rediscache.EventInvalidateAdminCache: admin.ResetCache,
		rediscache.EventRuntimesInstall:      admin.InstallDependencies,
	}

	for event, handler := range handlers {
		if err := service.SubscribeAsync(event, handler); err != nil {
			slog.Errorf("failed to register event %s: %v", event, err)
		}
	}

	mux := asynq.NewServeMux()

	mux.HandleFunc(tasks.DeploymentStart, HandleDeploymentStart)
	mux.HandleFunc(tasks.TriggerFunctionHttp, HandleFunctionTrigger)

	concurrency := 10

	if runner := config.Get().Runner; runner != nil && runner.Concurrency > 0 {
		concurrency = runner.Concurrency
	}

	slog.Infof("worker server up and running with concurrency=%d", concurrency)

	return tasks.NewServer(map[string]int{
		tasks.QueueDeployServicePriority: 3,
		tasks.QueueApiWebWS:              2,
		tasks.QueueDeployService:         1,
	}, concurrency), mux
}
