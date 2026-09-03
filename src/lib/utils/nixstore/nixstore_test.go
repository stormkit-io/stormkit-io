package nixstore_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore/nixstoretest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sys/unix"
)

type NixStoreSuite struct {
	suite.Suite
	mockCommand  *mocks.CommandInterface
	originalPath string
	restore      func()
}

func (s *NixStoreSuite) SetupSuite() {
	s.originalPath = nixstore.DefaultPath
}

func (s *NixStoreSuite) BeforeTest(_, _ string) {
	s.mockCommand = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCommand
	nixstore.DefaultPath = s.T().TempDir()
	s.restore = nixstoretest.StubLookPath(true)
}

func (s *NixStoreSuite) AfterTest(_, _ string) {
	sys.DefaultCommand = nil
	nixstore.DefaultPath = s.originalPath
	s.restore()
}

func (s *NixStoreSuite) Test_Available() {
	s.True(nixstore.Available())
}

func (s *NixStoreSuite) Test_Available_WithoutNixBinary() {
	s.restore()
	s.restore = nixstoretest.StubLookPath(false)

	s.False(nixstore.Available())
}

func (s *NixStoreSuite) Test_Available_WithoutStore() {
	nixstore.DefaultPath = "/does-not-exist"

	s.False(nixstore.Available())
}

func (s *NixStoreSuite) Test_DiskUsage() {
	usage, err := nixstore.DiskUsage(nixstore.DefaultPath)

	s.NoError(err)
	s.Equal(nixstore.DefaultPath, usage.Path)
	s.Greater(usage.TotalBytes, uint64(0))
	s.Equal(usage.TotalBytes-usage.FreeBytes, usage.UsedBytes)
	s.GreaterOrEqual(usage.UsedPercent, float64(0))
	s.LessOrEqual(usage.UsedPercent, float64(100))
}

func (s *NixStoreSuite) Test_DiskUsage_UnknownPath() {
	_, err := nixstore.DiskUsage("/does-not-exist")

	s.Error(err)
}

func (s *NixStoreSuite) Test_CollectGarbage() {
	s.mockCommand.On("SetOpts", sys.CommandOpts{
		Name: "nix-collect-garbage",
		Args: []string{"--delete-older-than", "3d"},
	}).Return(s.mockCommand).Once()

	s.mockCommand.On("CombinedOutput").Return([]byte(""), nil).Once()

	s.NoError(nixstore.CollectGarbage(context.Background(), nixstore.CollectGarbageParams{RetentionDays: 3}))
	s.mockCommand.AssertExpectations(s.T())
}

func (s *NixStoreSuite) Test_CollectGarbage_DefaultsRetention() {
	s.mockCommand.On("SetOpts", sys.CommandOpts{
		Name: "nix-collect-garbage",
		Args: []string{"--delete-older-than", "7d"},
	}).Return(s.mockCommand).Once()

	s.mockCommand.On("CombinedOutput").Return([]byte(""), nil).Once()

	s.NoError(nixstore.CollectGarbage(context.Background(), nixstore.CollectGarbageParams{}))
	s.mockCommand.AssertExpectations(s.T())
}

// Containers without Nix must not report an error: the same code path runs in
// images that never install it.
func (s *NixStoreSuite) Test_CollectGarbage_WithoutNix() {
	s.restore()
	s.restore = nixstoretest.StubLookPath(false)

	s.NoError(nixstore.CollectGarbage(context.Background(), nixstore.CollectGarbageParams{}))
	s.mockCommand.AssertNotCalled(s.T(), "CombinedOutput")
}

// Collecting before the roots are settled would delete an environment's
// packages while it still owns them, or keep a superseded generation alive.
func (s *NixStoreSuite) Test_CollectGarbage_SettlesRootsFirst() {
	originalProfiles := nixstore.ProfilesDir
	nixstore.ProfilesDir = filepath.Join(s.T().TempDir(), "stormkit")

	defer func() { nixstore.ProfilesDir = originalProfiles }()

	s.Require().NoError(nixstore.EnsureProfilesDir())

	stale := filepath.Join(nixstore.ProfilesDir, "app-1-env-1")
	s.Require().NoError(os.Symlink("/nix/store/fake-profile", stale+"-1-link"))
	s.Require().NoError(os.Symlink(stale+"-1-link", stale))

	old := time.Now().AddDate(0, 0, -30)
	ts := unix.NsecToTimespec(old.UnixNano())
	s.Require().NoError(unix.UtimesNanoAt(unix.AT_FDCWD, stale, []unix.Timespec{ts, ts}, unix.AT_SYMLINK_NOFOLLOW))

	var calls []string

	s.mockCommand.On("SetOpts", mock.Anything).Return(s.mockCommand).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(0).(sys.CommandOpts).Name)
	})

	s.mockCommand.On("CombinedOutput").Return([]byte(""), nil)

	s.NoError(nixstore.CollectGarbage(context.Background(), nixstore.CollectGarbageParams{RetentionDays: 7}))

	s.Equal([]string{"nix-env", "nix-collect-garbage"}, calls)
	s.NoFileExists(stale)
}

func (s *NixStoreSuite) Test_CollectGarbage_ReportsOutput() {
	s.mockCommand.On("SetOpts", mock.Anything).Return(s.mockCommand).Once()
	s.mockCommand.On("CombinedOutput").Return([]byte("cannot open lock file"), errCommandFailed).Once()

	err := nixstore.CollectGarbage(context.Background(), nixstore.CollectGarbageParams{})

	s.Error(err)
	s.Contains(err.Error(), "cannot open lock file")
}

func TestNixStoreSuite(t *testing.T) {
	suite.Run(t, new(NixStoreSuite))
}
