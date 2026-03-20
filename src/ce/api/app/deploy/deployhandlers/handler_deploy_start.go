package deployhandlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"gopkg.in/guregu/null.v3"
)

// handlerDeployStart starts the deployment process for the given app.
// This handler is triggered when the user submits a deploy request
// through the user interface.
func handlerDeployStart(req *app.RequestContext) *shttp.Response {
	if strings.Contains(req.Header.Get("content-type"), "multipart/form-data") {
		return deployZip(req)
	}

	var data = &deploy.RequestData{}
	var err error

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	if env == nil {
		return shttp.NotFound()
	}

	depl := deploy.New(req.App)
	depl.PopulateFromEnv(env)

	depl.Branch = utils.GetString(data.Branch, req.App.DefaultBranch())
	depl.ShouldPublish = data.Publish

	if err = deployservice.New().Deploy(req.Context(), req.App, depl); err != nil {
		if err == oauth.ErrRepoNotFound || err == oauth.ErrCredsInvalidPermissions {
			return &shttp.Response{
				Status: http.StatusNotFound,
				Data: map[string]string{
					"error": "Repository is not found or is inaccessible.",
				},
			}
		}

		if err == deployservice.ErrBuildMinutesExceeded {
			return &shttp.Response{
				Status: http.StatusPaymentRequired,
				Data: map[string]string{
					"error": "You have exceeded your build minutes limit. Please upgrade your plan to continue building your projects.",
				},
			}
		}

		return shttp.Error(err)
	}

	return &shttp.Response{
		Data:  depl,
		Error: err,
	}
}

const maxUploadSize = 10 << 20     // 10 MB
const uploadMemoryLimit = 10 << 20 // 10 MB

func deployZip(req *app.RequestContext) *shttp.Response {
	if config.IsStormkitCloud() && req.ContentLength > maxUploadSize {
		return shttp.BadRequest(map[string]any{
			"error": "You can upload maximum 100MB at a time.",
		})
	}

	if err := req.ParseMultipartForm(uploadMemoryLimit); err != nil {
		return shttp.BadRequest(map[string]any{
			"error": err.Error(),
		})
	}

	tmpDir, res := uploadFile(req)

	// Make sure to remove the tmpDir after processing
	if tmpDir != "" || (res != nil && res.Status != http.StatusOK) {
		defer os.RemoveAll(tmpDir)
	}

	if res != nil {
		return res
	}

	zipFilePath := path.Join(tmpDir, "sk-client.zip")
	unzipDir := path.Join(tmpDir, "app")

	if err := os.MkdirAll(unzipDir, os.ModePerm); err != nil {
		return shttp.Error(err)
	}

	unzipOpts := file.UnzipOpts{
		ZipFile:    zipFilePath,
		ExtractDir: unzipDir,
		LowerCase:  false, // Keep original case for files to construct manifest with original names
	}

	if err := file.Unzip(unzipOpts); err != nil {
		return shttp.BadRequest(map[string]any{
			"error": "The uploaded file is either not a zip file, or it is corrupt.",
		})
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	manifest := &deploy.BuildManifest{}
	manifest.Redirects, err = deploy.ParseRedirects([]string{
		path.Join(unzipDir, utils.GetString(env.Data.RedirectsFile, "redirects.json")),
		path.Join(unzipDir, "_redirects"), // Netlify-style
	})

	if err != nil {
		return shttp.BadRequest(map[string]any{
			"error": fmt.Sprintf("Cannot parse redirects file: %s", err.Error()),
		})
	}

	headersFile := utils.GetString(env.Data.HeadersFile, "_headers")
	headers, err := deploy.ParseHeadersFile(path.Join(unzipDir, headersFile))

	if err != nil {
		return shttp.BadRequest(map[string]any{
			"error": fmt.Sprintf("Cannot parse headers file: %s", headersFile),
		})
	}

	if env.Data.ServerCmd == "" {
		manifest.StaticFiles = deploy.PrepareStaticFiles([]string{unzipDir}, headers)
	}

	d := &deploy.Deployment{
		AppID:         req.App.ID,
		EnvID:         req.EnvID,
		Env:           env.Name,
		DisplayName:   req.App.DisplayName,
		BuildManifest: manifest,
		IsAutoDeploy:  false,
		ShouldPublish: req.FormValue("publish") == "true",
		Commit: deploy.CommitInfo{
			Author: null.StringFrom(req.User.Display()),
		},
	}

	store := deploy.NewStore()

	if err := store.InsertDeployment(req.Context(), d); err != nil {
		return shttp.Error(err)
	}

	args := runner.UploadArgs{
		EnvVars:      env.Data.Vars,
		Runtime:      req.App.Runtime,
		DeploymentID: d.ID,
		AppID:        req.App.ID,
		EnvID:        req.EnvID,
	}

	if env.Data.ServerCmd != "" {
		args.ServerZip = zipFilePath
	} else {
		args.ClientZip = zipFilePath
	}

	result, err := runner.NewUploader(config.Get().Runner).Upload(args)

	if err != nil {
		return shttp.Error(err)
	}

	d.IsImmutable = null.BoolFrom(true)
	d.ExitCode = null.IntFrom(0)

	if err := store.UpdateDeploymentResult(req.Context(), d, *result); err != nil {
		return shttp.Error(err)
	}

	if d.ShouldPublish {
		if err := deploy.AutoPublishIfNecessary(req.Context(), d); err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Data:  d.JSON(true),
		Error: err,
	}
}

func uploadFile(req *app.RequestContext) (string, *shttp.Response) {
	fileID := req.Header.Get("X-File-ID")

	if fileID == "" {
		return "", shttp.BadRequest(map[string]any{
			"error": "Missing X-File-ID header.",
		})
	}

	// Create a temporary directory
	tmpDir := path.Join(os.TempDir(), fmt.Sprintf("u-%s", fileID))

	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return "", shttp.Error(err)
	}

	files := req.MultipartForm.File["files"]

	if len(files) != 1 {
		return "", shttp.BadRequest(map[string]any{
			"error": "Please upload only one zip file.",
		})
	}

	return handleChunkedUpload(req, tmpDir, files[0])
}

// handleChunkedUpload processes chunked uploads.
func handleChunkedUpload(req *app.RequestContext, tmpDir string, file *multipart.FileHeader) (string, *shttp.Response) {
	// Parse metadata from headers
	chunkIndex := utils.StringToInt(req.Header.Get("X-Chunk-Index"))
	totalChunks := utils.GetInt(utils.StringToInt(req.Header.Get("X-Total-Chunks")), 1)

	// Save the chunk to a temporary file
	chunkFilePath := filepath.Join(tmpDir, fmt.Sprintf("chunk-%d", chunkIndex))
	chunkFile, err := os.Create(chunkFilePath)

	if err != nil {
		return "", shttp.Error(err)
	}

	defer chunkFile.Close()

	uploadedChunk, err := file.Open()

	if err != nil {
		return "", shttp.Error(err)
	}

	defer uploadedChunk.Close()

	if _, err := io.Copy(chunkFile, uploadedChunk); err != nil {
		return "", shttp.Error(err)
	}

	// List all uploaded chunks for debugging purposes
	chunks, err := os.ReadDir(tmpDir)

	if err != nil {
		return "", shttp.Error(err)
	}

	// Check if all chunks are uploaded
	if len(chunks) == totalChunks {
		if err := assembleChunks(tmpDir, totalChunks); err != nil {
			return "", shttp.Error(err)
		}

		return tmpDir, nil
	}

	return "", &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"message": fmt.Sprintf("Chunk %d uploaded successfully", chunkIndex),
		},
	}
}

// assembleChunks combines all chunks into the final file.
func assembleChunks(chunkDir string, totalChunks int) error {
	finalFilePath := filepath.Join(chunkDir, "sk-client.zip")
	finalFile, err := os.Create(finalFilePath)

	if err != nil {
		return err
	}

	defer finalFile.Close()

	for i := range totalChunks {
		chunkFilePath := filepath.Join(chunkDir, fmt.Sprintf("chunk-%d", i))
		chunkFile, err := os.Open(chunkFilePath)

		if err != nil {
			return err
		}

		_, err = io.Copy(finalFile, chunkFile)
		chunkFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
