package sysstats

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

// procSelfStatm is Linux's per-process memory summary. It is absent on other
// platforms, where the Go runtime figures are used instead.
const procSelfStatm = "/proc/self/statm"

// ProcessStats describes the Stormkit process reporting them, as opposed to the
// machine it runs on. The machine figures answer "is the box loaded"; these
// answer "is it Stormkit's fault".
type ProcessStats struct {
	Service    string `json:"service"`
	InstanceID string `json:"instanceId"`

	Goroutines int    `json:"goroutines"`
	HeapBytes  uint64 `json:"heapBytes"`

	// RSSBytes is resident set size. Zero when the platform does not expose it,
	// in which case the UI falls back to HeapBytes.
	RSSBytes uint64 `json:"rssBytes"`
}

type CollectProcessParams struct {
	Service    string
	InstanceID string
}

// CollectProcess reads this process's own resource usage. It is cheap enough to
// call on the registration heartbeat.
func CollectProcess(p CollectProcessParams) ProcessStats {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return ProcessStats{
		Service:    p.Service,
		InstanceID: p.InstanceID,
		Goroutines: runtime.NumGoroutine(),
		HeapBytes:  mem.HeapAlloc,
		RSSBytes:   residentBytes(),
	}
}

// residentBytes reads current RSS from /proc/self/statm, whose second field is
// the resident page count. Getrusage was avoided deliberately: it reports peak
// RSS, which never falls and would misrepresent a process that has since freed
// memory.
func residentBytes() uint64 {
	data, err := os.ReadFile(procSelfStatm)

	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))

	if len(fields) < 2 {
		return 0
	}

	pages, err := strconv.ParseUint(fields[1], 10, 64)

	if err != nil {
		return 0
	}

	return pages * uint64(os.Getpagesize())
}
