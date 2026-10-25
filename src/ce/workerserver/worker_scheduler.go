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

	// masterCancel cancels the context of the current leadership term, so
	// in-flight master runs stop when the leadership is renounced.
	masterCancel context.CancelFunc

	// masterTaskDefs overrides the master task list; used by tests.
	masterTaskDefs []TaskDefinition
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

// RegisterMasterTasks registers tasks that should be executed only by one
// node. Any master tasks registered earlier are removed first, so calling it
// again never runs a task twice. The tasks run with a context derived from
// ctx that is cancelled by StopMasterTasks.
func (s *Scheduler) RegisterMasterTasks(ctx context.Context) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.removeMasterTasks()

	termCtx, cancel := context.WithCancel(ctx)
	s.masterCancel = cancel
	s.masterTasks = s.registerTasks(termCtx, s.masterTaskDefinitions())
}

func (s *Scheduler) masterTaskDefinitions() []TaskDefinition {
	if s.masterTaskDefs != nil {
		return s.masterTaskDefs
	}

	immediate := []gocron.JobOption{
		gocron.JobOption(gocron.WithStartImmediately()),
	}

	dj := gocron.DurationJob
	daily := gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0)))

	return []TaskDefinition{
		{Handler: InvokeDueFunctionTriggers, Def: dj(EVERY_MINUTE), Opt: immediate},
		{Handler: RemoveOldLogs, Def: dj(EVERY_HOUR * 2), Opt: immediate},
		{Handler: RemoveOldAnalytics, Def: dj(EVERY_6_HOURS), Opt: immediate},
		{Handler: MaintainAccessLogPartitions, Def: daily, Opt: immediate},
		{Handler: RemoveStaleEnvironments, Def: dj(EVERY_6_HOURS), Opt: immediate},
		{Handler: EnsureAuthSchemas, Def: daily, Opt: immediate},
		{Handler: RemoveDeploymentArtifacts, Def: dj(EVERY_6_HOURS), Opt: immediate},
		{Handler: SyncAnalyticsVisitorsHourly, Def: dj(EVERY_MINUTE * 5), Opt: immediate},
		// Every daily aggregate is rebuilt in one hourly run, so reports read
		// one consistent snapshot and raw page views are scanned per day once.
		{Handler: SyncAnalyticsRollups, Def: dj(EVERY_HOUR), Opt: immediate},
		{Handler: CleanupDeletedTeams, Def: dj(EVERY_HOUR), Opt: immediate},
		{Handler: PingDomains, Def: dj(EVERY_MINUTE), Opt: immediate},
		{Handler: TimedOutDeployments, Def: dj(EVERY_MINUTE), Opt: immediate},
	}
}

// StopMasterTasks stops tasks that should be executed by the leader node.
func (s *Scheduler) StopMasterTasks() {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.removeMasterTasks()
}

// removeMasterTasks expects the caller to hold s.mux.
func (s *Scheduler) removeMasterTasks() {
	if s.masterCancel != nil {
		s.masterCancel()
		s.masterCancel = nil
	}

	for _, task := range s.masterTasks {
		s.scheduler.RemoveJob(task.ID())
	}

	s.masterTasks = []gocron.Job{}
}

func (s *Scheduler) registerTasks(ctx context.Context, tasks []TaskDefinition) []gocron.Job {
	registeredTasks := []gocron.Job{}

	for _, task := range tasks {
		// A run that outlasts its interval (e.g. a large analytics rollup)
		// skips the overlapping tick instead of running concurrently.
		opts := append([]gocron.JobOption{gocron.WithSingletonMode(gocron.LimitModeReschedule)}, task.Opt...)
		job, err := s.scheduler.NewJob(task.Def, gocron.NewTask(task.Handler, ctx), opts...)

		if err != nil {
			slog.Errorf("error while registering job: %s", err.Error())
		} else {
			registeredTasks = append(registeredTasks, job)
		}
	}

	return registeredTasks
}
