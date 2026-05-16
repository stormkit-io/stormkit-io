package publicapiv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/tasks"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerDeploymentPrioritize(req *RequestContext) *shttp.Response {
	id := utils.StringToID(req.Vars()["id"])

	if id == 0 {
		return shttp.NotFound()
	}

	store := deploy.NewStore()

	depl, err := store.MyDeployment(req.Context(), &deploy.DeploymentsQueryFilters{
		DeploymentID: id,
		EnvID:        req.Env.ID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if depl == nil {
		return shttp.NotFound()
	}

	if depl.Status() != "running" {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"Only queued deployments can be prioritized"},
		})
	}

	taskID := fmt.Sprintf("deployment-%s", id.String())
	inspector := tasks.Inspector()

	info, err := inspector.GetTaskInfo(tasks.QueueDeployService, taskID)

	if errors.Is(err, asynq.ErrTaskNotFound) || errors.Is(err, asynq.ErrQueueNotFound) {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"Deployment is not queued or has already started"},
		})
	}

	if err != nil {
		return shttp.Error(err)
	}

	if info.State == asynq.TaskStateActive {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"Deployment is already being executed and cannot be reordered"},
		})
	}

	if _, err := tasks.Enqueue(req.Context(), tasks.DeploymentStart, string(info.Payload), &tasks.EnqueueOptions{
		MaxRetry:  10,
		QueueName: tasks.QueueDeployServicePriority,
		TaskID:    taskID,
	}); err != nil {
		return shttp.Error(err)
	}

	if err := inspector.DeleteTask(tasks.QueueDeployService, taskID); err != nil {
		return shttp.Error(err)
	}

	if err := store.PrioritizeDeployment(req.Context(), id); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"ok": true},
	}
}
