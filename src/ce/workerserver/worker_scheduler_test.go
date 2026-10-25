package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stretchr/testify/suite"
)

type WorkerSchedulerSuite struct {
	suite.Suite
}

// Test_RegisterMasterTasks_Twice verifies that a second leader election on the
// same node replaces the master tasks instead of adding another copy.
func (s *WorkerSchedulerSuite) Test_RegisterMasterTasks_Twice() {
	scheduler, err := jobs.NewScheduler()
	s.Require().NoError(err)

	ctx := context.Background()

	scheduler.RegisterMasterTasks(ctx)
	count := scheduler.JobCount()

	s.Greater(count, 0)
	s.Equal(count, scheduler.MasterTaskCount())

	scheduler.RegisterMasterTasks(ctx)

	s.Equal(count, scheduler.JobCount())
	s.Equal(count, scheduler.MasterTaskCount())
}

func (s *WorkerSchedulerSuite) Test_StopMasterTasks() {
	scheduler, err := jobs.NewScheduler()
	s.Require().NoError(err)

	ctx := context.Background()

	scheduler.RegisterReplicaTasks(ctx)
	replicas := scheduler.JobCount()

	scheduler.RegisterMasterTasks(ctx)
	scheduler.StopMasterTasks()

	s.Equal(replicas, scheduler.JobCount())
	s.Equal(0, scheduler.MasterTaskCount())
}

// Test_StopMasterTasks_CancelsInFlightRuns verifies that a master run still in
// progress is cancelled when the leadership is renounced.
func (s *WorkerSchedulerSuite) Test_StopMasterTasks_CancelsInFlightRuns() {
	scheduler, err := jobs.NewScheduler()
	s.Require().NoError(err)

	started := make(chan struct{})
	finished := make(chan struct{})

	scheduler.SetMasterTasks([]jobs.TaskDefinition{
		{
			Def: gocron.DurationJob(time.Hour),
			Opt: []gocron.JobOption{gocron.JobOption(gocron.WithStartImmediately())},
			Handler: func(ctx context.Context) error {
				close(started)
				<-ctx.Done()
				close(finished)
				return nil
			},
		},
	})

	scheduler.RegisterMasterTasks(context.Background())
	scheduler.Start()

	defer func() { s.NoError(scheduler.Shutdown()) }()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		s.Require().Fail("master task did not start")
	}

	scheduler.StopMasterTasks()

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		s.Fail("master task was not cancelled")
	}
}

func TestWorkerSchedulerSuite(t *testing.T) {
	suite.Run(t, &WorkerSchedulerSuite{})
}
