package integrations

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type UploadOverview struct {
	BytesUploaded int64
	FilesUploaded int64
	Location      string
}

type UploadResult struct {
	Client     UploadOverview
	Server     UploadOverview
	API        UploadOverview
	Migrations UploadOverview
}

type UploadArgs struct {
	DistDir       string            // The destination dir, in case we're using filesys
	ClientZip     string            // The path to the client zip.
	ServerZip     string            // The path to the server zip.
	APIZip        string            // The path to the api zip.
	MigrationsZip string            // The path to the migrations zip.
	ServerHandler string            // The handler name in the following format: index.js:handler
	APIHandler    string            // The handler name in the following format: index.js:handler
	EnvID         types.ID          // The Environment ID which this deployment belongs to.
	AppID         types.ID          // The Application ID which this deployment belongs to.
	EnvVars       map[string]string // The environment variables that are going to used in this deployment.
	DeploymentID  types.ID          // The ID of the deployment.
	Runtime       string            // The build runtime such as: nodejs12.x
	BucketName    string            // When provided, it will overwrite the default bucket name.
	FilePath      string            // When provided, this file will be uploaded to the bucket.

	// These are used internally
	funcType string
	handler  string
	zip      string
}

type File struct {
	Size         int64
	FullPath     string
	RelativePath string
	Pointer      multipart.File
	Content      []byte
	ContentType  string
}

type uploadFunc func(File, any) error

func FilePathWalkDir(root string) []File {
	var files []File

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)

		if err != nil {
			return err
		}

		files = append(files, File{
			Size:         info.Size(),
			FullPath:     path,
			Content:      content,
			RelativePath: strings.Split(path, root)[1][1:],
			ContentType:  DetectContentType(path, content),
		})

		return nil
	})

	if err != nil {
		slog.Errorf("error while walking files: %v", err)
	}

	return files
}

// DetectContentType tries to detect the content type.
// It uses the extension to figure out the type, if not successful
// uses the built-in http.DetectContentType method.
func DetectContentType(filePath string, content any) string {
	fileType := mime.TypeByExtension(path.Ext(filePath))

	if fileType == "" {
		switch v := content.(type) {
		case []byte:
			fileType = http.DetectContentType(v)
		case multipart.FileHeader:
			if contentType := v.Header.Get("Content-Type"); contentType != "" {
				return contentType
			}

			// More reliable way - detect from actual content
			file, err := v.Open()

			if err != nil {
				return ""
			}

			defer file.Close()

			// Read first 512 bytes for type detection
			buffer := make([]byte, 512)
			n, err := file.Read(buffer)

			if err != nil && err != io.EOF {
				return ""
			}

			buffer = buffer[:n]

			// Detect content type
			fileType = http.DetectContentType(buffer)

		default:
			return ""
		}

	}

	return fileType
}

// Upload runs the integration by walking over the artifacts and
// uploading them to the end destination.
func Upload(buildFolder string, uploadFile uploadFunc, args any) (UploadOverview, error) {
	result := UploadOverview{
		FilesUploaded: 0,
		BytesUploaded: 0,
	}

	var wg sync.WaitGroup
	var err error

	maxGoroutines := config.Get().Runner.MaxGoRoutines

	// Create a channel to control the number of active goroutines
	// The channel has a buffer size equal to the maximum allowed goroutines
	semaphore := make(chan struct{}, maxGoroutines)

	for _, file := range FilePathWalkDir(buildFolder) {
		// Acquire a semaphore slot
		semaphore <- struct{}{}

		wg.Add(1)

		go func(f File) {
			defer func() {
				// Release the semaphore slot when the goroutine is done
				<-semaphore
				// Decrement the wait group counter
				wg.Done()
			}()

			if err := uploadFile(f, args); err != nil {
				slog.Error(err)
			} else {
				result.FilesUploaded = result.FilesUploaded + 1
				result.BytesUploaded = result.BytesUploaded + f.Size
			}
		}(file)
	}

	wg.Wait()

	return result, err
}

// parseHeaders will take a map of string/any and return a map of string/string.
func parseHeaders(headers map[string]any) http.Header {
	var _headers map[string]string

	if headers != nil {
		_headers = map[string]string{}

		for k, v := range headers {
			switch val := v.(type) {
			case string:
				_headers[k] = val
			case []any:
				headers[k] = ""
				for _, vany := range val {
					if vstr, ok := vany.(string); ok {
						_headers[k] = _headers[k] + vstr
					}
				}
			case []string:
				_headers[k] = strings.Join(val, ",")
			default:
				continue
			}
		}
	}

	return shttp.HeadersFromMap(_headers)
}

func prepareInvokeRequest(args InvokeArgs) FunctionRequest {
	headers := map[string]string{}
	rawHeaders := []string{}

	for k, v := range args.Headers {
		headers[k] = strings.Join(v, ",")
		rawHeaders = append(rawHeaders, k, headers[k])
	}

	if args.Headers.Get("Host") == "" {
		headers["host"] = args.HostName
	}

	var body []byte

	if args.Body != nil {
		body, _ = io.ReadAll(args.Body)
	}

	var relativeUrl string

	// The node.js specs suggests `request.url` to be a relative url with the query string.
	// However, nitro-based apps require the url to be a full URL.
	if strings.HasSuffix(args.ARN, "stormkit-api.mjs:handler") {
		relativeUrl = args.URL.Path

		if args.URL.RawQuery != "" {
			relativeUrl = relativeUrl + "?" + args.URL.RawQuery
		}
	} else {
		relativeUrl = args.URL.String()
	}

	return FunctionRequest{
		CaptureLogs: args.CaptureLogs,
		Method:      args.Method,
		Path:        args.URL.Path,
		Query:       args.URL.Query(),
		Headers:     headers,
		RawHeaders:  rawHeaders,
		URL:         relativeUrl,
		Body:        string(body),
		Context:     args.Context,
	}
}
