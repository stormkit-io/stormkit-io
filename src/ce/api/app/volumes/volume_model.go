package volumes

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	FileSys    = "filesys"
	AWSS3      = "aws:s3"
	AlibabaOSS = "alibaba:oss"
	HCloudOSS  = "hcloud:oss"
)

var (
	MaxUploadSize     = int64(50 << 20)  // 50 MB
	UploadMemoryLimit = int64(100 << 20) // 100 MB
)

func init() {
	if v := os.Getenv("STORMKIT_VOLUMES_MAX_UPLOAD_SIZE"); v != "" {
		if size, err := strconv.ParseInt(v, 10, 64); err == nil {
			MaxUploadSize = size
		}
	}

	if v := os.Getenv("STORMKIT_VOLUMES_UPLOAD_MEMORY_LIMIT"); v != "" {
		if size, err := strconv.ParseInt(v, 10, 64); err == nil {
			UploadMemoryLimit = size
		}
	}
}

type File struct {
	ID        types.ID
	EnvID     types.ID
	Name      string // The file name with the maintained structure (e.g. test-folder/file.txt)
	Path      string // The absolute path to the file including relative path (/shared/volumes/test-folder)
	Size      int64
	IsPublic  bool
	CreatedAt utils.Unix
	UpdatedAt utils.Unix
	Metadata  utils.Map
}

// FullPath returns the absolute path of the file.
func (f *File) FullPath() string {
	return filepath.Join(f.Path, filepath.Base(f.Name))
}

// PublicLink returns the link to access the file.
func (f *File) PublicLink() string {
	token := utils.EncryptToString(f.ID.String() + ":" + f.EnvID.String())
	return admin.MustConfig().ApiURL(fmt.Sprintf("/volumes/file/%s", token))
}

type UploadArgs struct {
	AppID              types.ID
	EnvID              types.ID
	FileHeader         FileHeader
	ContentDisposition map[string]string
}

// LimitRequestBody returns a middleware that wraps the request body with
// http.MaxBytesReader so that multipart parsing (e.g. triggered inside
// WithAPIKey via req.FormValue) cannot buffer an unbounded body to disk.
// For multipart requests the form is parsed eagerly so that an oversized
// body is caught here and returns a 413 with a clear message, rather than
// surfacing as a nil-dereference later in the handler.
func LimitRequestBody() shttp.RequestFunc {
	return func(req *shttp.RequestContext) *shttp.Response {
		if req.Body == nil || req.Writer() == nil {
			return nil
		}

		req.Body = http.MaxBytesReader(req.Writer(), req.Body, MaxUploadSize)

		if !strings.HasPrefix(strings.ToLower(req.Header.Get("Content-Type")), "multipart/form-data") {
			return nil
		}

		if err := req.ParseMultipartForm(UploadMemoryLimit); err != nil {
			var maxBytesErr *http.MaxBytesError

			if errors.As(err, &maxBytesErr) {
				return &shttp.Response{
					Status: http.StatusRequestEntityTooLarge,
					Data: map[string]string{
						"error": fmt.Sprintf("Request body too large. You can upload up to %dMB at a time.", maxBytesErr.Limit/(1024*1024)),
					},
				}
			}

			return &shttp.Response{
				Status: http.StatusBadRequest,
				Data: map[string]string{
					"error": err.Error(),
				},
			}
		}

		return nil
	}
}

// SanitizeUploadFilename validates and normalizes an uploaded filename to prevent
// path traversal attacks that could write files outside the intended volume root.
func SanitizeUploadFilename(name string) (string, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return "", fmt.Errorf("filename must not be empty")
	}

	cleaned := filepath.Clean(name)

	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	if cleaned == "." || cleaned == ".." {
		return "", fmt.Errorf("invalid filename")
	}

	sep := string(filepath.Separator)

	if strings.HasPrefix(cleaned, ".."+sep) || strings.Contains(cleaned, sep+".."+sep) {
		return "", fmt.Errorf("path traversal segments are not allowed")
	}

	return cleaned, nil
}

// Upload a file to the destination specified by args.MountType.
func Upload(vc *admin.VolumesConfig, args UploadArgs) (*File, error) {
	file, err := args.FileHeader.Open()

	if err != nil {
		return nil, err
	}

	defer file.Close()

	fileName, err := SanitizeUploadFilename(utils.GetString(args.ContentDisposition["filename"], args.FileHeader.Name()))

	if err != nil {
		return nil, err
	}

	filePath := constructFilePath(args.AppID, args.EnvID)

	opts := uploadArgs{
		file:        file,
		size:        args.FileHeader.Size(),
		envID:       args.EnvID,
		dstFileName: fileName,
		dstFilePath: filePath,
	}

	switch vc.MountType {
	case FileSys:
		return clientFilesys(vc).upload(opts)
	case AWSS3:
		return clientAWS(vc).upload(opts)
	}

	return nil, nil
}

// Download downloads a file from the source.
func Download(vc *admin.VolumesConfig, file *File) (io.ReadSeeker, error) {
	switch vc.MountType {
	case FileSys:
		return clientFilesys(vc).download(file)
	case AWSS3:
		return clientAWS(vc).download(file)
	}

	return nil, nil
}

// Upload a file to the destination specified by args.MountType.
// This function returns a list of files that are successfully removed.
// When encountered an error other than os.IsNotExist, returns immediately.
func RemoveFiles(vc *admin.VolumesConfig, files []*File) ([]*File, error) {
	switch vc.MountType {
	case FileSys:
		return clientFilesys(vc).removeFiles(files)
	case AWSS3:
		return clientAWS(vc).removeFiles(files)
	}

	return nil, nil
}

func constructFilePath(appID, envID types.ID) string {
	return path.Join(fmt.Sprintf("a%se%s", appID, envID))
}

type FileHeader interface {
	Open() (multipart.File, error)
	Name() string
	Size() int64
}

type fileHeader struct {
	original *multipart.FileHeader
}

func (fh *fileHeader) Open() (multipart.File, error) {
	return fh.original.Open()
}

func (fh *fileHeader) Size() int64 {
	return fh.original.Size
}

func (fh *fileHeader) Name() string {
	return fh.original.Filename
}

func FromFileHeader(file *multipart.FileHeader) FileHeader {
	return &fileHeader{
		original: file,
	}
}
