package integrations

import (
	"container/list"
	"os"
	"strings"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	defaultFileCacheBudget  = 256 << 20 // 256 MiB
	defaultFileCacheMaxItem = 1 << 20   // 1 MiB
)

// fileCache keeps deployment files in memory so serving one does not read from
// disk and allocate a fresh copy on every request.
//
// Without it a busy host grows its heap with the number of in-flight requests,
// and that growth comes out of the kernel's page cache, which turns reads that
// were free into real disk I/O and slows everything further.
//
// Entries are immutable: a deployment id names one fixed set of files, so a
// new deployment writes new keys rather than invalidating old ones. Callers
// share the stored value and MUST NOT modify its content.
type fileCache struct {
	mu      sync.Mutex
	budget  int64
	maxItem int64
	used    int64
	entries map[string]*list.Element
	order   *list.List // front is most recently used
}

type fileCacheEntry struct {
	key  string
	file *GetFileResult
	size int64
}

func newFileCache() *fileCache {
	return &fileCache{
		budget:  int64(utils.StringToInt(os.Getenv("STORMKIT_FILE_CACHE_BYTES"))),
		maxItem: int64(utils.StringToInt(os.Getenv("STORMKIT_FILE_CACHE_MAX_FILE_BYTES"))),
		entries: map[string]*list.Element{},
		order:   list.New(),
	}
}

// configure applies defaults for anything not set in the environment. A budget
// of zero disables the cache, which is the way to turn it off without a
// release.
func (fc *fileCache) configure() *fileCache {
	if os.Getenv("STORMKIT_FILE_CACHE_BYTES") == "" {
		fc.budget = defaultFileCacheBudget
	}

	if fc.maxItem <= 0 {
		fc.maxItem = defaultFileCacheMaxItem
	}

	return fc
}

func (fc *fileCache) get(key string) (*GetFileResult, bool) {
	if fc == nil || fc.budget <= 0 {
		return nil, false
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()

	element, ok := fc.entries[key]

	if !ok {
		return nil, false
	}

	fc.order.MoveToFront(element)

	return element.Value.(*fileCacheEntry).file, true
}

// dropDeployment removes every entry belonging to a deployment.
//
// Without it the only way out of the cache is eviction pressure, so a
// deployment nobody serves any more holds its share of the budget until
// something else needs the room.
func (fc *fileCache) dropDeployment(deploymentID string) {
	if fc == nil {
		return
	}

	prefix := deploymentID + ":"

	fc.mu.Lock()
	defer fc.mu.Unlock()

	for key, element := range fc.entries {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		fc.used -= element.Value.(*fileCacheEntry).size
		fc.order.Remove(element)
		delete(fc.entries, key)
	}
}

// put stores a file, evicting least recently used entries to stay inside the
// budget. Files over the per-item cap are not stored: one large asset must not
// be able to push the whole working set out.
func (fc *fileCache) put(key string, file *GetFileResult) {
	if fc == nil || fc.budget <= 0 || file == nil {
		return
	}

	size := int64(len(file.Content))

	if size > fc.maxItem {
		return
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()

	if element, ok := fc.entries[key]; ok {
		fc.used -= element.Value.(*fileCacheEntry).size
		fc.order.Remove(element)
		delete(fc.entries, key)
	}

	for fc.used+size > fc.budget {
		oldest := fc.order.Back()

		if oldest == nil {
			return
		}

		entry := oldest.Value.(*fileCacheEntry)
		fc.used -= entry.size
		fc.order.Remove(oldest)
		delete(fc.entries, entry.key)
	}

	fc.entries[key] = fc.order.PushFront(&fileCacheEntry{key: key, file: file, size: size})
	fc.used += size
}
