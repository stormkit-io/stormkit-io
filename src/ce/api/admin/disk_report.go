package admin

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
)

// rootPath is the filesystem a container is most likely to exhaust. It is a
// variable so tests can point it at a temporary directory.
var rootPath = "/"

// diskReportInterval is how often each container publishes its disk usage.
// It is a variable so tests do not have to wait.
var diskReportInterval = time.Minute

// diskReportTTL outlives a few missed intervals, so a container that is busy
// or briefly unreachable does not vanish from the admin view. A container that
// is genuinely gone drops out on its own.
var diskReportTTL = 5 * time.Minute

// DiskReport is one container's view of the disk it runs on. Reports are
// per-container because hosting and workerserver mount separate /nix volumes.
type DiskReport struct {
	ServiceID   string          `json:"serviceId"`
	ServiceName string          `json:"serviceName"`
	Root        nixstore.Usage  `json:"root"`
	NixStore    *nixstore.Usage `json:"nixStore,omitempty"`
	ReportedAt  time.Time       `json:"reportedAt"`
}

// CurrentDiskReport measures the disk of the container it is called in. The
// Nix store is reported separately from the root filesystem because on a Swarm
// host the two are usually distinct mounts, and the store is the part an
// operator can actually reclaim.
func CurrentDiskReport() (DiskReport, error) {
	root, err := nixstore.DiskUsage(rootPath)

	if err != nil {
		return DiskReport{}, err
	}

	report := DiskReport{Root: root, ReportedAt: time.Now().UTC()}

	if nixstore.Available() {
		if usage, err := nixstore.DiskUsage(nixstore.DefaultPath); err == nil {
			report.NixStore = &usage
		}
	}

	return report, nil
}

// ReportDiskUsage publishes this container's disk usage so the admin UI can
// read it without shelling into the host. It is best effort: a container that
// cannot measure or publish its disk simply does not appear in the report.
func ReportDiskUsage(ctx context.Context) {
	report, err := CurrentDiskReport()

	if err != nil {
		slog.Errorf("could not read disk usage of %s: %v", rootPath, err)
		return
	}

	payload, err := json.Marshal(report)

	if err != nil {
		slog.Errorf("could not encode disk report: %v", err)
		return
	}

	client := rediscache.Client()

	if client == nil {
		return
	}

	key := rediscache.Service().Key(rediscache.KEY_DISK_USAGE)

	if err := client.Set(ctx, key, string(payload), diskReportTTL).Err(); err != nil {
		slog.Errorf("could not publish disk report: %v", err)
	}
}

// DiskReports collects what every registered container last published, newest
// service name first so the order is stable across polls.
func DiskReports() ([]DiskReport, error) {
	services := []string{
		rediscache.ServiceHosting,
		rediscache.ServiceWorkerserver,
	}

	registered, err := rediscache.Service().List(services)

	if err != nil {
		return nil, err
	}

	values, err := rediscache.GetAll(rediscache.KEY_DISK_USAGE, services)

	if err != nil {
		return nil, err
	}

	reports := []DiskReport{}

	for _, service := range registered {
		value := values[service.ID]

		if value == "" {
			continue
		}

		report := DiskReport{}

		if err := json.Unmarshal([]byte(value), &report); err != nil {
			slog.Errorf("could not decode disk report of %s: %v", service.ID, err)
			continue
		}

		report.ServiceID = service.ID
		report.ServiceName = service.Name

		reports = append(reports, report)
	}

	sort.Slice(reports, func(i, j int) bool {
		if reports[i].ServiceName == reports[j].ServiceName {
			return reports[i].ServiceID < reports[j].ServiceID
		}

		return reports[i].ServiceName < reports[j].ServiceName
	})

	return reports, nil
}
