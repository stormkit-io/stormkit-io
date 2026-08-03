package sysstats

import (
	"io"
	"sort"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// virtualFSTypes are filesystems that occupy no disk. node_exporter already
// drops most of these, but overlay and tmpfs still come through in a container
// host and would otherwise show up as phantom disks in the UI.
var virtualFSTypes = map[string]bool{
	"tmpfs":      true,
	"devtmpfs":   true,
	"ramfs":      true,
	"squashfs":   true,
	"overlay":    true,
	"autofs":     true,
	"fuse.lxcfs": true,
}

// cpuCounters holds the previous scrape's CPU totals so the next one can turn
// node_cpu_seconds_total (a counter) into a percentage.
type cpuCounters struct {
	idle  float64
	total float64
}

// parser turns a node_exporter exposition response into a Sample.
type parser struct {
	families map[string]*dto.MetricFamily
}

func newParser(r io.Reader) (*parser, error) {
	// The zero TextParser leaves its validation scheme unset, which panics on
	// the first metric name. It has to be constructed with an explicit scheme.
	tp := expfmt.NewTextParser(model.UTF8Validation)

	families, err := tp.TextToMetricFamilies(r)

	if err != nil {
		return nil, err
	}

	return &parser{families: families}, nil
}

// value reads a metric's value regardless of how it is typed. node_exporter
// mixes gauges and counters, and untyped shows up behind some proxies.
func (p *parser) value(m *dto.Metric) float64 {
	if m.GetGauge() != nil {
		return m.GetGauge().GetValue()
	}

	if m.GetCounter() != nil {
		return m.GetCounter().GetValue()
	}

	if m.GetUntyped() != nil {
		return m.GetUntyped().GetValue()
	}

	return 0
}

func (p *parser) metrics(name string) []*dto.Metric {
	family, ok := p.families[name]

	if !ok {
		return nil
	}

	return family.GetMetric()
}

// scalar returns the value of a single-series metric such as node_load1.
func (p *parser) scalar(name string) float64 {
	for _, m := range p.metrics(name) {
		return p.value(m)
	}

	return 0
}

func (p *parser) label(m *dto.Metric, name string) string {
	for _, l := range m.GetLabel() {
		if l.GetName() == name {
			return l.GetValue()
		}
	}

	return ""
}

// cpu sums idle and total CPU seconds across every core and mode.
func (p *parser) cpu() (counters cpuCounters, cores int) {
	seen := map[string]bool{}

	for _, m := range p.metrics("node_cpu_seconds_total") {
		v := p.value(m)
		counters.total += v

		if p.label(m, "mode") == "idle" {
			counters.idle += v
		}

		if cpu := p.label(m, "cpu"); cpu != "" && !seen[cpu] {
			seen[cpu] = true
			cores++
		}
	}

	return counters, cores
}

// filesystems pairs node_filesystem_size_bytes with node_filesystem_avail_bytes
// by mountpoint. Both are reported per mount, so the two families are joined
// rather than read independently.
func (p *parser) filesystems() []Filesystem {
	byMount := map[string]*Filesystem{}

	for _, m := range p.metrics("node_filesystem_size_bytes") {
		fsType := p.label(m, "fstype")

		if virtualFSTypes[fsType] {
			continue
		}

		mount := p.label(m, "mountpoint")

		byMount[mount] = &Filesystem{
			Mountpoint: mount,
			Device:     p.label(m, "device"),
			FSType:     fsType,
			SizeBytes:  uint64(p.value(m)),
		}
	}

	for _, m := range p.metrics("node_filesystem_avail_bytes") {
		if fs, ok := byMount[p.label(m, "mountpoint")]; ok {
			fs.AvailBytes = uint64(p.value(m))
		}
	}

	out := make([]Filesystem, 0, len(byMount))

	for _, fs := range byMount {
		// A zero-size filesystem carries no information and clutters the panel.
		if fs.SizeBytes == 0 {
			continue
		}

		out = append(out, *fs)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Mountpoint < out[j].Mountpoint
	})

	return out
}

// network sums traffic across real interfaces. Loopback and virtual bridges
// would otherwise double-count container traffic against the host's.
func (p *parser) network(name string) uint64 {
	var total float64

	for _, m := range p.metrics(name) {
		device := p.label(m, "device")

		if device == "lo" || strings.HasPrefix(device, "veth") || strings.HasPrefix(device, "br-") || device == "docker0" {
			continue
		}

		total += p.value(m)
	}

	return uint64(total)
}

// sample builds a Sample from the parsed families. prev is the previous
// scrape's CPU counters, or nil on the first scrape of a target.
func (p *parser) sample(prev *cpuCounters) (*Sample, cpuCounters) {
	counters, cores := p.cpu()

	s := &Sample{
		Reachable:         true,
		CPUCores:          cores,
		MemTotalBytes:     uint64(p.scalar("node_memory_MemTotal_bytes")),
		MemAvailableBytes: uint64(p.scalar("node_memory_MemAvailable_bytes")),
		Load1:             p.scalar("node_load1"),
		Load5:             p.scalar("node_load5"),
		Load15:            p.scalar("node_load15"),
		Filesystems:       p.filesystems(),
		NetReceiveBytes:   p.network("node_network_receive_bytes_total"),
		NetTransmitBytes:  p.network("node_network_transmit_bytes_total"),
		BootTime:          int64(p.scalar("node_boot_time_seconds")),
	}

	s.CPUPercent, s.CPUValid = cpuPercent(prev, counters)

	return s, counters
}

// cpuPercent derives busy percentage from two counter readings. It reports
// invalid on the first sample and on counter reset (an exporter restart), so a
// restart shows a gap rather than a negative spike.
func cpuPercent(prev *cpuCounters, cur cpuCounters) (float64, bool) {
	if prev == nil {
		return 0, false
	}

	totalDelta := cur.total - prev.total
	idleDelta := cur.idle - prev.idle

	if totalDelta <= 0 || idleDelta < 0 {
		return 0, false
	}

	percent := (1 - idleDelta/totalDelta) * 100

	if percent < 0 {
		percent = 0
	}

	if percent > 100 {
		percent = 100
	}

	return percent, true
}
