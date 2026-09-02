package deployservice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"go.uber.org/zap"
)

type localService struct {
	tempDir    string // The temporary directory that the deployment will be checked out and built
	executable string // The path to stormkit-deployer-node repository
	apiWebUrl  string
}

// Local creates a new client to communicate with Local deploy service.
func Local() DeployerService {
	conf := config.Get()

	return &localService{
		executable: conf.Deployer.Executable,
		apiWebUrl:  admin.MustConfig().DomainConfig.API,
	}
}

// SendPayload starts the deployment process by sending the payload to the local deployer service.
func (ls *localService) SendPayload(payload SendPayloadArgs) error {
	if err := ls.prepareTempDir(payload.DeploymentID.String()); err != nil {
		return err
	}

	return ls.runService(payload)
}

func (ls *localService) StopDeployment(runID int64) error {
	return nil
}

func (ls *localService) prepareTempDir(deploymentID string) error {
	tmpDir := path.Join(os.TempDir(), fmt.Sprintf("deployment-%s", deploymentID))

	if err := os.MkdirAll(tmpDir, 0776); err != nil {
		return err
	}

	subdirs := []string{"repo", "keys"}
	slog.Infof("preparing folders: %s, tmp dir: %s", strings.Join(subdirs, ", "), tmpDir)

	for _, subdir := range subdirs {
		dir := filepath.Join(tmpDir, subdir)

		// If the directory already exists, remove it and create a new one
		if file.Exists(dir) {
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
		}

		if err := os.MkdirAll(dir, 0776); err != nil {
			return err
		}
	}

	ls.tempDir = tmpDir
	return nil
}

func (ls *localService) runService(args SendPayloadArgs) error {
	payload := map[string]any{
		"baseUrl":       ls.apiWebUrl,
		"rootDir":       ls.tempDir,
		"deploymentMsg": args.EncryptedMsg,
		"deploymentId":  args.DeploymentID.String(),
	}

	msg, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	deployerDir := config.Get().Deployer.StorageDir

	slog.Debug(slog.LogOpts{
		Msg:   "starting local deployer service",
		Level: slog.DL3,
		Payload: []zap.Field{
			zap.String("executable", ls.executable),
			zap.String("root_dir", ls.tempDir),
			zap.String("deployer_dir", deployerDir),
		},
	})

	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name:   ls.executable,
		Args:   []string{"--payload", string(msg)},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env: []string{
			fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
			fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
			fmt.Sprintf("STORMKIT_DEPLOYER_DIR=%s", deployerDir),
			fmt.Sprintf("STORMKIT_DEPLOYER_SERVICE=%s", config.DeployerServiceLocal),
			fmt.Sprintf("STORMKIT_APP_SECRET=%s", config.AppSecret()),
		},
	})

	return cmd.Run()
}
