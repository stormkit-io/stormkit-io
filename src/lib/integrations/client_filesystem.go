package integrations

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

type FilesysClient struct {
	pm  *ProcessManager
	mjs *template.Template
	cjs *template.Template
}

var _filesys *FilesysClient
var _filesysMux sync.Mutex

func Filesys() *FilesysClient {
	_filesysMux.Lock()
	defer _filesysMux.Unlock()

	if _filesys == nil {
		_filesys = &FilesysClient{
			mjs: template.Must(template.New("mjs").Parse(strings.Join(strings.Fields(`
				const log = (r) => r && console.log(JSON.stringify(r)) || process.exit(0);
				const err = (e) => console.log({ body: e && e.message ? e.message : e, status: 500 }) || process.exit(1);

				import("./{{ .fileName }}").then(m => {
					m.{{ .handlerName }}({{ .payload }}, {}, (e, r) => log(r))
						.then(log)
						.catch(err)
				}).catch(err)
			`), " "))),

			cjs: template.Must(template.New("cjs").Parse(strings.Join(strings.Fields(`
				require("./{{ .fileName }}").{{ .handlerName }}({{ .payload }}, {}, (e,r) => console.log(JSON.stringify(r)))
			`), " "))),
		}
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
		return c.ProcessManager().Invoke(args, path.Dir(fnPath))
	}

	requestPayload, err := json.Marshal(prepareInvokeRequest(args))

	if err != nil {
		return nil, err
	}

	var wr bytes.Buffer
	var tmpl *template.Template

	fileName := path.Base(fnPath)
	fileDir := path.Dir(fnPath)

	if strings.HasSuffix(fnPath, ".mjs") {
		tmpl = c.mjs
	} else {
		tmpl = c.cjs
	}

	err = tmpl.Execute(&wr, map[string]string{
		"fileName":    fileName,
		"handlerName": fnHandler,
		"payload":     string(requestPayload),
	})

	if err != nil {
		slog.Errorf("error while executing template: %v", err)
		return nil, err
	}

	vars := []string{}

	for k, v := range args.EnvVariables {
		vars = append(vars, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name: "node",
		Args: []string{"-e", wr.String()},
		Env:  vars,
		Dir:  fileDir,
	})

	out, err := cmd.CombinedOutput()

	if err != nil {
		slog.Errorf("error while running local command: %v", err)
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

// Upload a file to the file system. Use the DistDir argument to specify
// the destination folder.
func (c *FilesysClient) Upload(args UploadArgs) (result *UploadResult, err error) {
	dir := utils.GetString(args.DistDir, config.Get().Deployer.StorageDir)
	root := path.Join(dir, fmt.Sprintf("deployment-%d", args.DeploymentID))
	result = &UploadResult{}

	result.Client, err = c.uploadZip(uploadZipArgs{
		pathToZip:   args.ClientZip,
		targetDir:   path.Join(root, "client"),
		shouldUnzip: true,
	})

	if err != nil {
		return nil, err
	}

	result.Server, err = c.uploadZip(uploadZipArgs{
		pathToZip:   args.ServerZip,
		fileName:    args.ServerHandler,
		targetDir:   path.Join(root, "server"),
		shouldUnzip: true,
	})

	if err != nil {
		return nil, err
	}

	result.API, err = c.uploadZip(uploadZipArgs{
		pathToZip:   args.APIZip,
		fileName:    args.APIHandler,
		targetDir:   path.Join(root, "api"),
		shouldUnzip: true,
	})

	if err != nil {
		return nil, err
	}

	result.Migrations, err = c.uploadZip(uploadZipArgs{
		pathToZip:   args.MigrationsZip,
		targetDir:   path.Join(root, "migrations"),
		fileName:    path.Base(args.MigrationsZip), // Not really the handler, but a workaround to store the zip file name
		shouldUnzip: false,
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

type uploadZipArgs struct {
	pathToZip   string
	fileName    string
	targetDir   string
	shouldUnzip bool
}

// uploadZip uploads a zip file to the filesystem by unzipping it
// to the target directory.
func (c *FilesysClient) uploadZip(args uploadZipArgs) (UploadOverview, error) {
	if args.pathToZip == "" {
		return UploadOverview{}, nil
	}

	if err := os.MkdirAll(args.targetDir, 0774); err != nil {
		return UploadOverview{}, err
	}

	fstat, err := os.Stat(args.pathToZip)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return UploadOverview{}, nil
		}

		return UploadOverview{}, err
	}

	unzipOpts := file.UnzipOpts{
		ZipFile:    args.pathToZip,
		ExtractDir: args.targetDir,
		LowerCase:  false,
	}

	if args.shouldUnzip {
		if err := file.Unzip(unzipOpts); err != nil {
			return UploadOverview{}, err
		}
	} else {
		zipFileName := path.Base(args.pathToZip)
		destPath := path.Join(args.targetDir, zipFileName)

		if err := file.Copy(args.pathToZip, destPath, 0664); err != nil {
			return UploadOverview{}, err
		}
	}

	return UploadOverview{
		FilesUploaded: 1,
		BytesUploaded: fstat.Size(),
		Location:      fmt.Sprintf("local:%s", path.Join(args.targetDir, args.fileName)),
	}, nil
}

func (c *FilesysClient) parseFunctionLocation(location string) (string, string) {
	pieces := strings.Split(strings.TrimPrefix(location, "local:"), ":")

	if len(pieces) == 1 {
		return pieces[0] + "/.", ""
	}

	// location, handler
	return pieces[0], pieces[1]
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
