package tasks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
)

var (
	QueueDeployService         = "deploy-service"
	QueueDeployServicePriority = "deploy-service-priority"
	QueueApiWebWS              = "workerserver" // default queue
	Ping                       = "ping"
)

func init() {
	if config.IsTest() {
		QueueDeployService = "test-deploy-service"
		QueueDeployServicePriority = "test-deploy-service-priority"
	}
}

// A list of task types.
const (
	DeploymentStart     = "deployment:start"
	TriggerFunctionHttp = "triggerfunction:http"
)

type EnqueueOptions struct {
	MaxRetry  int
	QueueName string
	TaskID    string
}
type TaskClient interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type TaskInspector interface {
	GetTaskInfo(queue, taskID string) (*asynq.TaskInfo, error)
	DeleteTask(queue, taskID string) error
	DeleteQueue(queue string, force bool) error
}

var client *asynq.Client
var inspector *asynq.Inspector
var scheduler *asynq.Scheduler

// Client is a singleton method which creates a new asynq client and
// returns it in subsequent calls.
var Client = func() TaskClient {
	if client == nil {
		client = asynq.NewClient(asynq.RedisClientOpt{Addr: config.Get().RedisAddr})
	}

	return client
}

// Scheduler is a singleton method which creates a new asynq scheduler and
// returns it in subsequent calls.
func Scheduler() *asynq.Scheduler {
	if scheduler == nil {
		scheduler = asynq.NewScheduler(asynq.RedisClientOpt{Addr: config.Get().RedisAddr}, nil)
	}

	return scheduler
}

// NewServer returns a new server instance.
func NewServer(queues map[string]int, concurrency int) *asynq.Server {
	var logLevel asynq.LogLevel

	if config.IsTest() {
		logLevel = asynq.ErrorLevel
	}

	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: config.Get().RedisAddr},
		asynq.Config{
			LogLevel:       logLevel,
			Concurrency:    concurrency,
			Queues:         queues,
			StrictPriority: true,
		},
	)
}

// Inspector is a singleton method which creates a new asynq inspector and
// returns it in subsequent calls.
var Inspector = func() TaskInspector {
	if inspector == nil {
		inspector = asynq.NewInspector(asynq.RedisClientOpt{Addr: config.Get().RedisAddr})
	}

	return inspector
}

// Enqueue is a shortcut function for asynq.Client.Enqueue method.
func Enqueue(ctx context.Context, taskName string, message any, opts *EnqueueOptions) (*asynq.TaskInfo, error) {
	var payload []byte
	var err error

	if _, isString := message.(string); isString {
		payload = []byte(message.(string))
	} else {
		payload, err = json.Marshal(message)
	}

	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &EnqueueOptions{}
	}

	if opts.MaxRetry == 0 {
		opts.MaxRetry = 5
	}

	if opts.QueueName == "" {
		opts.QueueName = "workerserver"
	}

	options := []asynq.Option{
		asynq.Queue(opts.QueueName),
		asynq.MaxRetry(opts.MaxRetry),
	}

	if config.IsSelfHosted() {
		options = append(options, asynq.Deadline(time.Now().Add(time.Hour*6)))
	}

	if opts.TaskID != "" {
		options = append(options, asynq.TaskID(opts.TaskID))
	}

	return Client().Enqueue(asynq.NewTask(taskName, payload, options...))
}
