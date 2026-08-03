package tracking

import (
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
)

// DeploymentStorage reports free and total bytes on the filesystem holding
// deployment artifacts.
//
// node_exporter already covers every filesystem on the machine, but only
// Stormkit knows which one its deployments land on — that mapping is what makes
// this worth exporting separately.
//
// These are GaugeFuncs so the value is read at scrape time; there is no sampler
// to schedule and nothing to keep in memory between scrapes.
var (
	DeploymentStorageFree = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "stormkit",
			Subsystem: "storage",
			Name:      "deployments_free_bytes",
			Help:      "Free bytes on the filesystem holding deployment artifacts",
		},
		func() float64 {
			stat, err := statStorageDir()

			if err != nil {
				return 0
			}

			return float64(stat.Bavail) * float64(stat.Bsize)
		},
	)

	DeploymentStorageTotal = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "stormkit",
			Subsystem: "storage",
			Name:      "deployments_total_bytes",
			Help:      "Total bytes on the filesystem holding deployment artifacts",
		},
		func() float64 {
			stat, err := statStorageDir()

			if err != nil {
				return 0
			}

			return float64(stat.Blocks) * float64(stat.Bsize)
		},
	)
)

func statStorageDir() (*syscall.Statfs_t, error) {
	dir := config.Get().Deployer.StorageDir

	if dir == "" {
		return nil, syscall.ENOENT
	}

	var stat syscall.Statfs_t

	if err := syscall.Statfs(dir, &stat); err != nil {
		return nil, err
	}

	return &stat, nil
}
