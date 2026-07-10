package runner

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"go.uber.org/zap"
)

type ReporterModel struct {
	steps       *stepLogger
	baseURL     string
	CallbackURL string
	done        chan struct{}
	isDone      bool
	mux         sync.Mutex
}

// These are manipulated by external packages
var DeploymentIDEnc string
var FlagPrintLogs bool

func NewReporter(baseURL string) *ReporterModel {
	return &ReporterModel{
		steps:       newStepLogger(),
		done:        make(chan struct{}),
		baseURL:     baseURL,
		CallbackURL: baseURL + "/app/deploy/callback",
	}
}

func (r *ReporterModel) request(payload map[string]any) error {
	res, err := shttp.NewRequestV2(shttp.MethodPost, r.CallbackURL).
		WithExponentialBackoff(time.Second*30, 5).
		Headers(r.headers()).
		Payload(payload).Do()

	var body []byte

	if res != nil && res.Body != nil {
		body, err = io.ReadAll(res.Body)

		if err != nil {
			slog.Errorf("cannot read response body: %s", err.Error())
		}

		res.Body.Close()
	}

	if res == nil {
		return fmt.Errorf("no response received from deployment callback: %w", err)
	}

	if res.StatusCode == http.StatusConflict {
		slog.Info("received exit signal - quitting")
		os.Exit(128)
	}

	if res.StatusCode >= 400 {
		slog.Debug(slog.LogOpts{
			Msg: fmt.Sprintf("deployment callback returned status code %d - quitting", res.StatusCode),
			Payload: []zap.Field{
				zap.String("body", string(body)),
			},
		})

		os.Exit(128)
	}

	return err
}

// Logs returns the accumulated build logs in the structured NDJSON format.
func (r *ReporterModel) Logs() string {
	if r.steps == nil {
		return ""
	}

	return r.steps.marshal()
}

func (r *ReporterModel) headers() http.Header {
	headers := make(http.Header)
	headers.Add("Accept", "application/json")
	headers.Add("Content-Type", "application/json")
	return headers
}

func (r *ReporterModel) sendLogs() error {
	logs := r.Logs()

	if logs == "" {
		slog.Info("nothing to send, skipping this time")
		return nil
	}

	return r.request(map[string]any{
		"deployId": DeploymentIDEnc,
		"logs":     logs,
	})
}

func (r *ReporterModel) SendLogs() {
	if r.steps == nil || r.baseURL == "" {
		return
	}

	go func() {
		for {
			select {
			case <-r.done:
				// No need to send the logs one last time here because send exit already sends it
				return
			default:
				if err := r.sendLogs(); err != nil {
					slog.Errorf("error while sending logs: %s", err.Error())
				}

				time.Sleep(time.Second * 5)
			}
		}
	}()
}

func (r *ReporterModel) SendCommitInfo(info map[string]string) error {
	if r.baseURL == "" {
		return nil
	}

	// Wait until we have something
	if info["sha"] == "" {
		return nil
	}

	return r.request(map[string]any{
		"deployId": DeploymentIDEnc,
		"commit":   info,
		"runId":    os.Getenv("GITHUB_RUN_ID"),
	})
}

// LockDeployment should be called only after status checks are called.
// If a deployment has no status checks, the exit callback will lock
// the deployment automatically.
func (r *ReporterModel) LockDeployment(isSuccess bool) error {
	outcome := "failure"

	if isSuccess {
		outcome = "success"
	}

	// Send remaining logs
	r.sendLogs()

	return r.request(map[string]any{
		"deployId":        DeploymentIDEnc,
		"outcome":         outcome,
		"hasStatusChecks": true,
		"lock":            true,
	})
}

func (r *ReporterModel) SendExit(manifest *deploy.BuildManifest, result *integrations.UploadResult, hasStatusChecks bool, uploadErr error) error {
	if r.steps == nil || r.baseURL == "" {
		return nil
	}

	outcome := "failure"

	if manifest != nil && manifest.Success {
		outcome = "success"
	}

	// The step that was open when the build ended carries the outcome:
	// the deploy step on success, the failing step otherwise.
	r.steps.closeStep(outcome == "success")

	uploadErrStr := ""

	if uploadErr != nil {
		uploadErrStr = uploadErr.Error()
	}

	// Send last remaining bits
	r.sendLogs()

	// Start over for the status check steps
	r.steps = newStepLogger()

	err := r.request(map[string]any{
		"deployId":        DeploymentIDEnc,
		"outcome":         outcome,
		"manifest":        manifest,
		"error":           uploadErrStr,
		"result":          result,
		"hasStatusChecks": hasStatusChecks,
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *ReporterModel) AddStep(title string) {
	if r.steps == nil {
		return
	}

	r.steps.addStep(title)
}

func (r *ReporterModel) AddLine(text string) {
	if r.steps == nil {
		return
	}

	if _, err := r.steps.Write([]byte(fmt.Sprintf("%s\n", text))); err != nil {
		slog.Errorf("cannot add line: %s", text)
	}
}

// CloseStep closes the current step with an explicit outcome. Steps closed
// implicitly by the next AddStep are marked successful, so this is only
// needed when a step can fail without ending the build (e.g. status checks).
func (r *ReporterModel) CloseStep(success bool) {
	if r.steps == nil {
		return
	}

	r.steps.closeStep(success)
}

func (r *ReporterModel) Close(manifest *deploy.BuildManifest, result *integrations.UploadResult, err error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	if r.isDone {
		return
	}

	r.isDone = true
	close(r.done)

	if r.steps != nil {
		r.steps = nil
	}
}

// File returns the writer command output should stream into. The writer is
// stable across steps: output is attributed to whichever step is current at
// write time.
func (r *ReporterModel) File() io.Writer {
	if r.steps == nil {
		return io.Discard
	}

	return r.steps
}
