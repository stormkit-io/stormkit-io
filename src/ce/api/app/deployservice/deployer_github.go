package deployservice

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/go-github/v71/github"
	"github.com/pkg/errors"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"golang.org/x/oauth2"
)

var _ghService DeployerService

type ghService struct {
	*github.Client

	owner        string
	repo         string
	workflowFile string
}

// Github creates a new client to communicate with GitHub.
func Github() DeployerService {
	if _ghService == nil {
		cnf := admin.MustConfig().AuthConfig.Github
		owner, repo := oauth.ParseRepo(cnf.RunnerRepo)

		_ghService = &ghService{
			owner:        owner,
			repo:         repo,
			workflowFile: "deploy.yml",

			Client: github.NewClient(
				oauth2.NewClient(
					context.TODO(),
					oauth2.StaticTokenSource(
						&oauth2.Token{
							AccessToken: cnf.RunnerToken,
						},
					),
				),
			),
		}
	}

	return _ghService
}

// SendPayload send a new deployment request to GitHub.
func (gc *ghService) SendPayload(payload SendPayloadArgs) error {
	bytePayload, err := json.Marshal(map[string]any{
		"baseUrl":       admin.MustConfig().DomainConfig.API,
		"deploymentMsg": payload.EncryptedMsg,
		"deploymentId":  payload.DeploymentID.String(),
	})

	if err != nil {
		return err
	}

	res, err := gc.Client.Actions.CreateWorkflowDispatchEventByFileName(
		context.TODO(),
		gc.owner,
		gc.repo,
		gc.workflowFile,
		github.CreateWorkflowDispatchEventRequest{
			Ref: "main",
			Inputs: map[string]any{
				"payload": string(bytePayload),
			},
		})

	if err != nil {
		return err
	}

	if res != nil && res.StatusCode != http.StatusNoContent {
		return errors.New("response did not return 204")
	}

	return nil
}

// StopDeployment stops the workflow with the given runID.
func (gc *ghService) StopDeployment(runID int64) error {
	_, err := gc.Client.Actions.CancelWorkflowRunByID(context.Background(), gc.owner, gc.repo, runID)
	return err
}
