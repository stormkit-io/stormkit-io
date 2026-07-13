package volumes

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type uploadArgs struct {
	file        multipart.File
	size        int64
	dstFileName string
	dstFilePath string
	envID       types.ID
}

type vmFileSys struct {
	RootPath string
}

var CachedFileSys *vmFileSys
var mux sync.Mutex

// clientFilesys is a singleton function to return either the cached
// file system volume manager or create one from scratch.
func clientFilesys(c *admin.VolumesConfig) *vmFileSys {
	mux.Lock()
	defer mux.Unlock()

	if CachedFileSys == nil {
		CachedFileSys = &vmFileSys{
			RootPath: c.RootPath,
		}
	}

	return CachedFileSys
}

func (fs *vmFileSys) download(file *File) (io.ReadSeeker, error) {
	f, err := os.Open(file.FullPath())

	if os.IsNotExist(err) {
		return nil, nil
	}

	return f, nil
}

// upload a file to the destination specified by args.MountType.
// This function assumes that the file has already been sanitized and validated by the caller.
func (fs *vmFileSys) upload(args uploadArgs) (*File, error) {
	destination, err := filepath.Abs(
		filepath.Join(fs.RootPath, args.dstFilePath, filepath.Dir(args.dstFileName)),
	)

	if err != nil {
		return nil, err
	}

	// Make sure that the folder exists
	if err := os.MkdirAll(destination, 0775); err != nil {
		return nil, err
	}

	dst, err := os.Create(filepath.Join(destination, filepath.Base(args.dstFileName)))

	if err != nil {
		return nil, err
	}

	size, err := io.Copy(dst, args.file)

	if err != nil {
		return nil, err
	}

	return &File{
		Size:      size,
		Path:      destination,
		Name:      args.dstFileName,
		EnvID:     args.envID,
		CreatedAt: utils.NewUnix(),
		Metadata: utils.Map{
			"mountType": FileSys,
		},
	}, nil
}

func (fs *vmFileSys) removeFiles(files []*File) ([]*File, error) {
	success := []*File{}

	for _, file := range files {
		if err := os.Remove(file.FullPath()); err != nil {
			if os.IsNotExist(err) {
				success = append(success, file)
				continue
			}

			return success, err
		}

		success = append(success, file)
	}

	return success, nil
}
