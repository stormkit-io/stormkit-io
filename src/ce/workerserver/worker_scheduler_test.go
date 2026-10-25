package jobs_test

import (
	"context"
	"testing"

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

func TestWorkerSchedulerSuite(t *testing.T) {
	suite.Run(t, &WorkerSchedulerSuite{})
}
