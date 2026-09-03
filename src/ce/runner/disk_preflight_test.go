package runner_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore/nixstoretest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type DiskPreflightSuite struct {
	suite.Suite
	mockCommand  *mocks.CommandInterface
	originalPath string
	restore      func()
	dir          string
	reporter     *runner.ReporterModel
}

func (s *DiskPreflightSuite) check() error {
	return runner.DiskPreflightCheck(context.Background(), s.dir, s.reporter)
}

func (s *DiskPreflightSuite) BeforeTest(_, _ string) {
	s.mockCommand = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCommand

	s.originalPath = nixstore.DefaultPath
	nixstore.DefaultPath = s.T().TempDir()
	s.restore = nixstoretest.StubLookPath(false)

	s.dir = s.T().TempDir()
	s.reporter = runner.NewReporter("https://example.com")
}

func (s *DiskPreflightSuite) AfterTest(_, _ string) {
	sys.DefaultCommand = nil
	nixstore.DefaultPath = s.originalPath
	s.restore()
}

func (s *DiskPreflightSuite) Test_MinFreeBytes_Default() {
	s.Equal(uint64(runner.DefaultMinFreeDiskMB)<<20, runner.DiskPreflightMinFreeBytes())
}

func (s *DiskPreflightSuite) Test_MinFreeBytes_Override() {
	s.T().Setenv(runner.MinFreeDiskEnvVar, "512")

	s.Equal(uint64(512)<<20, runner.DiskPreflightMinFreeBytes())
}

func (s *DiskPreflightSuite) Test_MinFreeBytes_Disabled() {
	s.T().Setenv(runner.MinFreeDiskEnvVar, "0")

	s.Equal(uint64(0), runner.DiskPreflightMinFreeBytes())
}

// A typo in the env var must not silently disable the check.
func (s *DiskPreflightSuite) Test_MinFreeBytes_InvalidFallsBackToDefault() {
	s.T().Setenv(runner.MinFreeDiskEnvVar, "plenty")

	s.Equal(uint64(runner.DefaultMinFreeDiskMB)<<20, runner.DiskPreflightMinFreeBytes())
}

func (s *DiskPreflightSuite) Test_Check_Disabled() {
	s.T().Setenv(runner.MinFreeDiskEnvVar, "0")

	s.NoError(s.check())
}

func (s *DiskPreflightSuite) Test_Check_EnoughSpace() {
	s.T().Setenv(runner.MinFreeDiskEnvVar, "1")

	s.NoError(s.check())
}

// The check exists to give a better error, not to add a new way to fail: a
// build host that cannot report its own usage is allowed to proceed.
func (s *DiskPreflightSuite) Test_Check_UnreadablePathProceeds() {
	s.NoError(runner.DiskPreflightCheck(context.Background(), "/does-not-exist", s.reporter))
}

func (s *DiskPreflightSuite) Test_Check_NotEnoughSpace() {
	// Larger than any real disk, so the branch is deterministic.
	s.T().Setenv(runner.MinFreeDiskEnvVar, "1099511627776")

	err := s.check()

	s.Error(err)
	s.Contains(err.Error(), "Not enough disk space on the build host")
	s.Contains(err.Error(), "retry the deployment")
	s.mockCommand.AssertNotCalled(s.T(), "Run")
}

// When Nix is present the store is collected before giving up, since it is
// the usual reason the disk filled in the first place.
func (s *DiskPreflightSuite) Test_Check_CollectsNixGarbageBeforeFailing() {
	s.restore()
	s.restore = nixstoretest.StubLookPath(true)

	s.T().Setenv(runner.MinFreeDiskEnvVar, "1099511627776")

	s.mockCommand.On("SetOpts", sys.CommandOpts{
		Name: "nix-collect-garbage",
		Args: []string{"--delete-older-than", "7d"},
	}).Return(s.mockCommand).Once()

	s.mockCommand.On("Run").Return(nil).Once()

	err := s.check()

	s.Error(err)
	s.mockCommand.AssertExpectations(s.T())
	s.Contains(s.reporter.Logs(), "Reclaiming space from the Nix store")
}

func (s *DiskPreflightSuite) Test_HumanBytes() {
	s.Equal("512MB", runner.HumanBytes(512<<20))
	s.Equal("2.0GB", runner.HumanBytes(2<<30))
	s.Equal("0MB", runner.HumanBytes(0))
}

func TestDiskPreflightSuite(t *testing.T) {
	suite.Run(t, new(DiskPreflightSuite))
}
