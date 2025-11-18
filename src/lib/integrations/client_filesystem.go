package integrations

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

type FilesysClient struct {
	pm *ProcessManager
}

var _filesys *FilesysClient
var _filesysMux sync.Mutex

func Filesys() *FilesysClient {
	_filesysMux.Lock()
	defer _filesysMux.Unlock()

	if _filesys == nil {
		_filesys = &FilesysClient{}
	}

	return _filesys
}

func (c *FilesysClient) Name() string {
	return "Filesys"
}

func (c *FilesysClient) ProcessManager() *ProcessManager {
	_filesysMux.Lock()
	defer _filesysMux.Unlock()

	if c.pm == nil {
		c.pm = NewProcessManager()
	}

	return c.pm
}

func (c *FilesysClient) Invoke(args InvokeArgs) (*InvokeResult, error) {
	fnPath, fnHandler := c.parseFunctionLocation(args.ARN)

	if args.Command != "" {
		return c.ProcessManager().Invoke(args, filepath.Dir(fnPath))
	}

	requestPayload, err := json.Marshal(prepareInvokeRequest(args))

	if err != nil {
		return nil, err
	}

	var script string

	fileName := filepath.Base(fnPath)
	fileDir := filepath.Dir(fnPath)

	if strings.HasSuffix(fnPath, ".mjs") {
		script = fmt.Sprintf(`import("./%s").then(m => m.%s(%s, {}, (e, r) => console.log(JSON.stringify(r))).then(r => r && console.log(JSON.stringify(r))))`, fileName, fnHandler, string(requestPayload))
	} else {
		script = fmt.Sprintf(`require("./%s").%s(%s, {}, (e,r) => console.log(JSON.stringify(r)))`, fileName, fnHandler, string(requestPayload))
	}

	vars := []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
	}

	// Add NODE_PATH to help Node.js find dependencies
	// Look for node_modules in the fileDir and parent directories
	nodeModulesPath := filepath.Join(fileDir, "node_modules")
	if _, err := os.Stat(nodeModulesPath); err == nil {
		vars = append(vars, fmt.Sprintf("NODE_PATH=%s", nodeModulesPath))
	} else {
		// Try parent directory
		parentDir := filepath.Dir(fileDir)
		nodeModulesPath = filepath.Join(parentDir, "node_modules")
		if _, err := os.Stat(nodeModulesPath); err == nil {
			vars = append(vars, fmt.Sprintf("NODE_PATH=%s", nodeModulesPath))
		}
	}

	for k, v := range args.EnvVariables {
		vars = append(vars, fmt.Sprintf("%s=%s", k, v))
	}

	fmt.Println("DEBUG fnPath:", fnPath)
	fmt.Println("DEBUG fileName:", fileName)
	fmt.Println("DEBUG fileDir:", fileDir)
	fmt.Println("DEBUG script:", script)

	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name: "node",
		Args: []string{"-e", script},
		Env:  vars,
		Dir:  fileDir,
	})

	out, err := cmd.CombinedOutput()

	if err != nil {
		slog.Errorf("error while running local command: %v, output: %s", err, string(out))
		return nil, err
	}

	if out == nil {
		return nil, nil
	}

	response := FunctionResponse{}

	if err := json.Unmarshal(out, &response); err != nil {
		return nil, err
	}

	body := utils.GetString(response.Buffer, response.Body)

	invokeResult := &InvokeResult{
		Logs:         response.Logs,
		Body:         []byte(body),
		Headers:      parseHeaders(response.Headers),
		StatusCode:   utils.GetInt(response.Status, response.StatusCode, http.StatusOK),
		ErrorMessage: response.ErrorMessage,
		ErrorStack:   response.ErrorStack,
	}

	// See if this is a base64 encoded string
	if decoded, err := base64.StdEncoding.DecodeString(body); err == nil {
		invokeResult.Body = decoded
	}

	return invokeResult, nil
}

// DeleteArtifacts deletes all artifacts associated with the deployment from the file system.
func (c *FilesysClient) DeleteArtifacts(ctx context.Context, args DeleteArtifactsArgs) error {
	// The FilesysClient stores files under a folder called `deployment-<deployment-id>` such as:
	//
	// <path>/deployment-29/server/.next:server
	// <path>/deployment-29/api/stormkit-api.mjs:handler
	// <path>/deployment-29/client
	//
	// To delete artifacts, it's enough to delete the parent folder.
	location := utils.GetString(args.StorageLocation, args.FunctionLocation, args.APILocation)

	// Nothing to delete
	if location == "" {
		return nil
	}

	return os.RemoveAll(c.getDeploymentPath(location))
}

// Upload a file to the file system. Use the DistDir argument to specify
// the destination folder.
func (c *FilesysClient) Upload(args UploadArgs) (*UploadResult, error) {
	var err error
	result := &UploadResult{}

	dir := args.DistDir

	if dir == "" {
		dir = config.Get().Deployer.StorageDir
	}

	depl := fmt.Sprintf("deployment-%d", args.DeploymentID)
	root := path.Join(dir, depl)

	if args.ClientZip != "" {
		copy := args
		copy.zip = args.ClientZip
		copy.handler = ""

		if result.Client, err = c.uploadZip(copy, path.Join(root, "client")); err != nil {
			return nil, err
		}
	}

	if args.ServerZip != "" {
		copy := args
		copy.zip = args.ServerZip
		copy.handler = args.ServerHandler

		if result.Server, err = c.uploadZip(copy, path.Join(root, "server")); err != nil {
			return nil, err
		}
	}

	if args.APIZip != "" {
		copy := args
		copy.zip = args.APIZip
		copy.handler = args.APIHandler

		if result.API, err = c.uploadZip(copy, path.Join(root, "api")); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// GetFile returns a file from the Filesystem.
func (c *FilesysClient) GetFile(args GetFileArgs) (*GetFileResult, error) {
	filePath := path.Join(strings.TrimPrefix(args.Location, "local:"), args.FileName)
	stat, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)

	if err != nil {
		return nil, err
	}

	return &GetFileResult{
		ContentType: DetectContentType(filePath, data),
		Size:        stat.Size(),
		Content:     data,
	}, nil
}

func (c *FilesysClient) uploadZip(args UploadArgs, to string) (UploadOverview, error) {
	if err := os.MkdirAll(to, 0774); err != nil {
		return UploadOverview{}, err
	}

	fstat, err := os.Stat(args.zip)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return UploadOverview{}, nil
		}

		return UploadOverview{}, err
	}

	unzipOpts := file.UnzipOpts{
		ZipFile:    args.zip,
		ExtractDir: to,
		LowerCase:  false,
	}

	if err := file.Unzip(unzipOpts); err != nil {
		return UploadOverview{}, err
	}

	return UploadOverview{
		BytesUploaded: fstat.Size(),
		FilesUploaded: 1,
		Location:      fmt.Sprintf("local:%s", path.Join(to, args.handler)),
	}, nil
}

func (c *FilesysClient) parseFunctionLocation(location string) (string, string) {
	fmt.Println("DEBUG parseFunctionLocation input:", location)

	// Remove the local: prefix
	location = strings.TrimPrefix(location, "local:")

	// On Windows, we need to handle the drive letter (C:) specially
	// The format is: path:handler or just path
	// Windows paths look like: C:\Users\...\file.mjs:handler

	// Find the last colon which should be the handler separator
	// But we need to skip the drive letter colon on Windows
	lastColon := strings.LastIndex(location, ":")

	// If no colon found, or it's just the drive letter (position 1 on Windows)
	if lastColon == -1 || (lastColon == 1 && len(location) > 2) {
		return location + "/.", ""
	}

	// If the colon is at position 1, it's a drive letter, look for another colon
	if lastColon == 1 {
		return location + "/.", ""
	}

	// Split into path and handler
	filePath := location[:lastColon]
	handler := location[lastColon+1:]

	fmt.Println("DEBUG parsed path:", filePath, "handler:", handler)

	return filePath, handler
}

// getDeploymentPath returns the deployment path from a location. The location
// can be a StorageLocation, FunctionLocation or APILocation.
func (a *FilesysClient) getDeploymentPath(location string) string {
	// Remove the `local:` prefix
	location = strings.TrimPrefix(location, "local:")
	i := 0

	for {
		// This is a fallback exit
		if i > 20 {
			return ""
		}

		base := path.Base(location)

		if strings.HasPrefix(base, "deployment-") {
			break
		}

		if base == "" {
			break
		}

		location = path.Dir(location)
		i = i + 1
	}

	return location
}
