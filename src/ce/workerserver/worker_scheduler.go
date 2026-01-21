package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

type TaskFunc func(context.Context) error

type TaskDefinition struct {
	Handler TaskFunc
	Def     gocron.JobDefinition
	Opt     []gocron.JobOption
}

const (
	EVERY_5_SECOND = time.Second * 5
	EVERY_MINUTE   = time.Minute
	EVERY_HOUR     = time.Hour
	EVERY_2_HOURS  = time.Hour * 2
	EVERY_6_HOURS  = time.Hour * 6
	EVERY_DAY      = time.Hour * 24
	EVERY_MONTH    = EVERY_DAY * 30
)

type Scheduler struct {
	scheduler    gocron.Scheduler
	replicaTasks []gocron.Job
	masterTasks  []gocron.Job
	mux          sync.Mutex
}

func NewScheduler() (*Scheduler, error) {
	s, err := gocron.NewScheduler()

	if err != nil {
		slog.Errorf("error while starting scheduler: %s", err)
		return nil, err
	}

	return &Scheduler{
		scheduler: s,
	}, nil
}

// Start starts the scheduler with the registered jobs.
func (s *Scheduler) Start() {
	s.scheduler.Start()
}

// RegisterReplicaTasks registers tasks that are safe to be executed by
// multiple workers.
func (s *Scheduler) RegisterReplicaTasks(ctx context.Context) {
	s.mux.Lock()
	defer s.mux.Unlock()

	tasks := []TaskDefinition{
		{Handler: IngestHandlerForward, Def: gocron.DurationJob(EVERY_5_SECOND)},
	}

	s.replicaTasks = s.registerTasks(ctx, tasks)
}

// RegisterMaster registers tasks that should be executed only by one node.
func (s *Scheduler) RegisterMasterTasks(ctx context.Context) {
	s.mux.Lock()
	defer s.mux.Unlock()

	immediate := []gocron.JobOption{
		gocron.JobOption(gocron.WithStartImmediately()),
	}

	dj := gocron.DurationJob
	daily := gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0)))

	tasks := []TaskDefinition{
		{Handler: InvokeDueFunctionTriggers, Def: dj(EVERY_MINUTE), Opt: immediate},
		{Handler: RemoveOldLogs, Def: dj(EVERY_HOUR * 2), Opt: immediate},
		{Handler: RemoveStaleEnvironments, Def: dj(EVERY_6_HOURS), Opt: immediate},
		{Handler: RemoveDeploymentArtifacts, Def: dj(EVERY_6_HOURS), Opt: immediate},
		{Handler: SyncAnalyticsVisitorsHourly, Def: dj(EVERY_MINUTE * 5), Opt: immediate},
		{Handler: SyncAnalyticsVisitorsDaily, Def: daily, Opt: immediate},
		{Handler: SyncAnalyticsReferrers, Def: dj(EVERY_HOUR), Opt: immediate},
		{Handler: SyncAnalyticsByCountries, Def: dj(EVERY_HOUR), Opt: immediate},
		{Handler: CleanupDeletedTeams, Def: dj(EVERY_HOUR), Opt: immediate},
		{Handler: PingDomains, Def: dj(EVERY_MINUTE), Opt: immediate},
		{Handler: TimedOutDeployments, Def: dj(EVERY_MINUTE), Opt: immediate},
	}

	s.masterTasks = s.registerTasks(ctx, tasks)
}

// StopMasterTasks stops tasks that should be executed the the leader node.
func (s *Scheduler) StopMasterTasks() {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, task := range s.masterTasks {
		s.scheduler.RemoveJob(task.ID())
	}

	s.masterTasks = []gocron.Job{}
}

func (s *Scheduler) registerTasks(ctx context.Context, tasks []TaskDefinition) []gocron.Job {
	registeredTasks := []gocron.Job{}

	for _, task := range tasks {
		job, err := s.scheduler.NewJob(task.Def, gocron.NewTask(task.Handler, ctx), task.Opt...)

		if err != nil {
			slog.Errorf("error while registering job: %s", err.Error())
		} else {
			registeredTasks = append(registeredTasks, job)
		}
	}

	return registeredTasks
}
