package integrations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"go.uber.org/zap"
)

const removeAfterInactivity = time.Hour * 6

type DownloadFunc func(deploymentID, bucketname, keyprefix string) (string, error)

type Zip struct {
	Location string
	cacheKey string
	timer    *time.Timer
	manager  *ZipManager
}

// NewZip creates a new Zip instance and attaches a timer that will remove
// the folder after the configured time.
func NewZip(key string, manager *ZipManager) *Zip {
	zip := &Zip{cacheKey: key, manager: manager}
	zip.timer = time.AfterFunc(removeAfterInactivity, zip.RemoveFolder)

	return zip
}

// RemoveFolder is the timer callback. It takes the per-deployment lock so it
// cannot race with a concurrent GetFile that is updating the timer or
// downloading. If a reader reset the timer between it firing and us
// acquiring the lock, Stop() returns true — put the timer back and skip the
// deletion. Otherwise CompareAndDelete + RemoveAll.
func (z *Zip) RemoveFolder() {
	lock := z.manager.didLock(z.cacheKey)
	lock.Lock()
	defer lock.Unlock()

	if z.timer.Stop() {
		z.timer.Reset(removeAfterInactivity)
		return
	}

	if !z.manager.cache.CompareAndDelete(z.cacheKey, z) {
		return
	}

	slog.Debug(slog.LogOpts{
		Msg:   "Removing folder after inactivity",
		Level: slog.DL2,
		Payload: []zap.Field{
			zap.String("folder", z.Location),
			zap.String("cacheKey", z.cacheKey),
		},
	})

	if z.Location != "" {
		if err := os.RemoveAll(z.Location); err != nil {
			slog.Errorf("error while removing zip folder: %s", err.Error())
		}
	}
}

type ZipManager struct {
	download DownloadFunc
	cache    sync.Map // deploymentID (string) -> *Zip
	// locks holds one *sync.Mutex per deployment id. Held during cache
	// check, optional download, and timer reset — released before the file
	// read so large reads don't block other readers of the same deployment.
	// Different deployments never contend with each other.
	//
	// Entries are intentionally never removed. Deleting a lock entry from
	// the map while another goroutine has already obtained the pointer is
	// unsafe: a later caller would see the entry missing and create a
	// second mutex for the same deployment id, ending up with two
	// goroutines that believe they are mutually excluded but are not.
	// Bounded cleanup would require reference-counted handles, which is
	// not worth the complexity at Stormkit's scale: each entry is roughly
	// 30 bytes (sync.Map node + Mutex), so the steady-state footprint is
	// well under 1 MB per ~30k distinct deployments seen by the process,
	// and process restarts on upgrade reset the map.
	locks sync.Map // deploymentID (string) -> *sync.Mutex
}

func NewZipManager(download DownloadFunc) *ZipManager {
	return &ZipManager{download: download}
}

func (zm *ZipManager) didLock(did string) *sync.Mutex {
	if v, ok := zm.locks.Load(did); ok {
		return v.(*sync.Mutex)
	}

	actual, _ := zm.locks.LoadOrStore(did, &sync.Mutex{})

	return actual.(*sync.Mutex)
}

func (zm *ZipManager) folderDoesNotExist(folder string) bool {
	if folder == "" {
		return true
	}

	_, err := os.Stat(folder)
	return os.IsNotExist(err)
}

// GetFile returns a file for the given deployment. The location is the fully
// qualified location name in the form "aws:<bucket>/<key-prefix>".
//
// Flow:
//  1. Acquire the per-deployment lock.
//  2. Check the cache; download the zip on a miss. Other readers of the
//     same deployment wait. Different deployments are independent.
//  3. Reset the inactivity timer.
//  4. Release the lock and read the file with no lock held — large reads
//     do not block other readers of the same deployment.
func (zm *ZipManager) GetFile(args GetFileArgs) (*GetFileResult, error) {
	did := args.DeploymentID.String()
	pieces := strings.Split(strings.TrimPrefix(args.Location, "aws:"), "/")
	bucket := pieces[0]
	prefix := filepath.Join(pieces[1:]...)

	lock := zm.didLock(did)
	lock.Lock()

	var z *Zip

	if v, ok := zm.cache.Load(did); ok {
		z = v.(*Zip)

		if zm.folderDoesNotExist(z.Location) {
			z = nil
		}
	}

	if z == nil {
		z = NewZip(did, zm)

		loc, err := zm.download(did, bucket, prefix)

		if err != nil {
			z.timer.Stop()
			lock.Unlock()
			return nil, err
		}

		z.Location = loc
		zm.cache.Store(did, z)
	}

	z.timer.Reset(removeAfterInactivity)
	location := z.Location
	lock.Unlock()

	filePath := filepath.Join(location, args.FileName)

	// Defense in depth: reject paths that escape the deployment directory.
	rel, err := filepath.Rel(location, filePath)

	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("file path %q escapes deployment directory", args.FileName)
	}

	data, err := os.ReadFile(filePath)

	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(filePath)

	if err != nil {
		return nil, err
	}

	return &GetFileResult{
		Content:     data,
		ContentType: DetectContentType(filePath, data),
		Size:        stat.Size(),
	}, nil
}
