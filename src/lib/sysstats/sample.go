// Package sysstats collects machine-level resource usage by scraping a
// node_exporter instance running on each host.
//
// Stormkit itself runs inside a container, where statfs only sees the
// filesystems mounted into that container. Getting full visibility would mean
// bind-mounting the host root into the hosting service, which also runs
// deployed applications as child processes — so node_exporter keeps that mount
// in a container of its own instead.
package sysstats

// Filesystem is a single mounted filesystem on the target machine.
type Filesystem struct {
	Mountpoint string `json:"mountpoint"`
	Device     string `json:"device"`
	FSType     string `json:"fsType,omitempty"`
	SizeBytes  uint64 `json:"sizeBytes"`
	AvailBytes uint64 `json:"availBytes"`
}

// UsedBytes is the space occupied on the filesystem. node_exporter reports size
// and available separately; available already excludes the blocks reserved for
// root, so this over-reports usage slightly on ext4. That matches what `df`
// shows, which is what an operator compares against.
func (f Filesystem) UsedBytes() uint64 {
	if f.AvailBytes > f.SizeBytes {
		return 0
	}

	return f.SizeBytes - f.AvailBytes
}

// Sample is a point-in-time snapshot of one machine.
type Sample struct {
	Timestamp int64  `json:"ts"`
	Target    string `json:"target"`

	// Reachable reports whether the exporter answered. An unreachable target is
	// still recorded, so the UI can say "node_exporter is not running here"
	// rather than silently omitting the machine.
	Reachable bool   `json:"reachable"`
	Error     string `json:"error,omitempty"`

	// CPUPercent is busy time across all cores over the interval since the
	// previous sample. CPUValid is false on the first sample of a target and
	// whenever the counter reset, since no meaningful rate exists then.
	CPUPercent float64 `json:"cpuPercent"`
	CPUValid   bool    `json:"cpuValid"`
	CPUCores   int     `json:"cpuCores"`

	MemTotalBytes     uint64 `json:"memTotalBytes"`
	MemAvailableBytes uint64 `json:"memAvailableBytes"`

	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`

	Filesystems []Filesystem `json:"filesystems"`

	NetReceiveBytes  uint64 `json:"netReceiveBytes"`
	NetTransmitBytes uint64 `json:"netTransmitBytes"`

	BootTime int64 `json:"bootTime"`
}

// MemUsedBytes is total minus available. Available (rather than free) is the
// figure that accounts for reclaimable page cache, so it reflects what an
// application could actually allocate.
func (s Sample) MemUsedBytes() uint64 {
	if s.MemAvailableBytes > s.MemTotalBytes {
		return 0
	}

	return s.MemTotalBytes - s.MemAvailableBytes
}
