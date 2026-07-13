package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhooks"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/tasks"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type WorkerServerSuite struct {
	suite.Suite
	*factory.Factory
	server            *asynq.Server
	mux               *asynq.ServeMux
	conn              databasetest.TestDB
	mockDeployService *mocks.DeployerService
	hooksCalled       bool
}

func (s *WorkerServerSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockDeployService = &mocks.DeployerService{}
	s.mockDeployService.On("SendPayload", mock.Anything).Return(nil)
	deployservice.SetMockService(s.mockDeployService)
	s.server, s.mux = jobs.Server(jobs.ServerOpts{Scheduler: false})

	// clean up left-over queue
	tasks.Inspector().DeleteQueue(tasks.QueueDeployService, true)

	deployhooks.Exec = func(ctx context.Context, d *deploy.Deployment) {
		s.hooksCalled = true
	}
}

func (s *WorkerServerSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()

	// This will ensure that all tasks are processed
	s.server.Stop()
	s.server.Shutdown()

	tasks.Inspector().DeleteQueue(tasks.QueueDeployService, true)
	deployservice.SetMockService(nil)
}

func (s *WorkerServerSuite) Test_TaskDeploymentStart() {
	depl := s.MockDeployment(nil).Deployment
	appl := s.GetApp().App

	// This creates a new task
	err := deployservice.New().Deploy(context.Background(), appl, depl)

	s.NoError(err)

	// Process the queue
	s.NoError(s.server.Start(s.mux))

	// This is ugly but it works - we need to wait a bit until the task
	// is picked up from the queue and processed.
	// This is how asynq does as well: https://github.com/hibiken/asynq/blob/master/processor_test.go#L139
	time.Sleep(100 * time.Millisecond)

	s.mockDeployService.AssertCalled(s.T(), "SendPayload", mock.Anything)
}

func TestWorkerServer(t *testing.T) {
	suite.Run(t, &WorkerServerSuite{})
}
