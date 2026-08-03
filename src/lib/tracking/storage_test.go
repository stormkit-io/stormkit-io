package tracking

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stretchr/testify/suite"
)

type StorageSuite struct {
	suite.Suite
}

func (s *StorageSuite) Test_ReportsFreeAndTotalBytes() {
	// StorageDir is relative in a dev config, so the test points it at a real
	// directory rather than depending on where the tests were invoked from.
	original := config.Get().Deployer.StorageDir
	defer func() { config.Get().Deployer.StorageDir = original }()

	config.Get().Deployer.StorageDir = s.T().TempDir()

	free := testutil.ToFloat64(DeploymentStorageFree)
	total := testutil.ToFloat64(DeploymentStorageTotal)

	s.Positive(total, "a real filesystem reports a size")
	s.GreaterOrEqual(total, free, "free space cannot exceed the disk")
}

// An unreadable directory yields zero rather than an error: a scrape must not
// fail because of one gauge.
func (s *StorageSuite) Test_MissingDirectoryYieldsZero() {
	original := config.Get().Deployer.StorageDir
	defer func() { config.Get().Deployer.StorageDir = original }()

	config.Get().Deployer.StorageDir = "/this/path/does/not/exist"

	s.Equal(float64(0), testutil.ToFloat64(DeploymentStorageFree))
	s.Equal(float64(0), testutil.ToFloat64(DeploymentStorageTotal))
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}
