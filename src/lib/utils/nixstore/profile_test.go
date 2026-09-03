package nixstore_test

import (
	"context"
	"errors"
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

// errCommandFailed stands in for the exit status a failed nix command returns.
var errCommandFailed = errors.New("exit status 1")

// lchtimes backdates a symlink without following it, which is what os.Chtimes
// would do: profiles are symlinks and their own timestamp is what marks a
// deployment.
func lchtimes(path string, modified time.Time) error {
	ts := unix.NsecToTimespec(modified.UnixNano())

	return unix.UtimesNanoAt(unix.AT_FDCWD, path, []unix.Timespec{ts, ts}, unix.AT_SYMLINK_NOFOLLOW)
}

type ProfileSuite struct {
	suite.Suite
	mockCommand      *mocks.CommandInterface
	originalStore    string
	originalProfiles string
	restore          func()
}

func (s *ProfileSuite) SetupSuite() {
	s.originalStore = nixstore.DefaultPath
	s.originalProfiles = nixstore.ProfilesDir
}

func (s *ProfileSuite) BeforeTest(_, _ string) {
	s.mockCommand = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCommand
	nixstore.DefaultPath = s.T().TempDir()
	nixstore.ProfilesDir = filepath.Join(s.T().TempDir(), "stormkit")
	s.restore = nixstoretest.StubLookPath(true)
}

func (s *ProfileSuite) AfterTest(_, _ string) {
	sys.DefaultCommand = nil
	nixstore.DefaultPath = s.originalStore
	nixstore.ProfilesDir = s.originalProfiles
	s.restore()
}

// writeProfile creates a profile and its newest generation link, backdated by
// age so retention can be exercised without waiting.
func (s *ProfileSuite) writeProfile(name string, age time.Duration) string {
	s.Require().NoError(nixstore.EnsureProfilesDir())

	profile := filepath.Join(nixstore.ProfilesDir, name)
	generation := profile + "-1-link"

	s.Require().NoError(os.Symlink("/nix/store/fake-profile", generation))
	s.Require().NoError(os.Symlink(generation, profile))

	// Both links dangle until Nix builds the closure they point at, and their
	// own timestamps are what mark the deployment, so neither may be followed.
	modified := time.Now().Add(-age)
	s.Require().NoError(lchtimes(generation, modified))
	s.Require().NoError(lchtimes(profile, modified))

	return profile
}

func (s *ProfileSuite) Test_ProfilePath() {
	s.Equal(
		filepath.Join(nixstore.ProfilesDir, "app-12-env-34"),
		nixstore.ProfilePath(nixstore.ProfileParams{AppID: "12", EnvID: "34"}),
	)
}

func (s *ProfileSuite) Test_ProfilePath_SanitizesIdentifiers() {
	path := nixstore.ProfilePath(nixstore.ProfileParams{AppID: "../../etc", EnvID: "3 4"})

	s.Equal(filepath.Join(nixstore.ProfilesDir, "app-etc-env-34"), path)
	s.Equal(nixstore.ProfilesDir, filepath.Dir(path))
}

// Without identifiers there is nothing to key a profile on. The caller falls
// back to an unrooted shell rather than sharing one profile between apps.
func (s *ProfileSuite) Test_ProfilePath_WithoutIdentifiers() {
	s.Empty(nixstore.ProfilePath(nixstore.ProfileParams{EnvID: "34"}))
	s.Empty(nixstore.ProfilePath(nixstore.ProfileParams{AppID: "12"}))
	s.Empty(nixstore.ProfilePath(nixstore.ProfileParams{AppID: "///", EnvID: "34"}))
}

func (s *ProfileSuite) Test_ListProfiles_IsEmptyWithoutDirectory() {
	profiles, err := nixstore.ListProfiles()

	s.NoError(err)
	s.Empty(profiles)
}

// Generation links belong to their profile and must not be listed as roots of
// their own, or reaping would count the same environment twice.
func (s *ProfileSuite) Test_ListProfiles_SkipsGenerationsAndSortsByNewest() {
	s.writeProfile("app-1-env-1", 10*24*time.Hour)
	s.writeProfile("app-2-env-2", time.Hour)

	profiles, err := nixstore.ListProfiles()

	s.NoError(err)
	s.Len(profiles, 2)
	s.Equal("app-2-env-2", profiles[0].Name)
	s.Equal("app-1-env-1", profiles[1].Name)
}

func (s *ProfileSuite) Test_RemoveProfile_RemovesGenerations() {
	profile := s.writeProfile("app-1-env-1", time.Hour)
	profiles, err := nixstore.ListProfiles()

	s.Require().NoError(err)
	s.Require().Len(profiles, 1)
	s.NoError(nixstore.RemoveProfile(profiles[0]))
	s.NoFileExists(profile)

	remaining, err := os.ReadDir(nixstore.ProfilesDir)

	s.NoError(err)
	s.Empty(remaining)
}

func (s *ProfileSuite) Test_ReapProfiles_DropsOnlyStaleEnvironments() {
	fresh := s.writeProfile("app-1-env-1", 2*24*time.Hour)
	stale := s.writeProfile("app-2-env-2", 30*24*time.Hour)

	reaped, err := nixstore.ReapProfiles(nixstore.ReapProfilesParams{RetentionDays: 7})

	s.NoError(err)
	s.Equal(1, reaped)
	s.NoFileExists(stale)

	_, err = os.Lstat(fresh)
	s.NoError(err)
}

func (s *ProfileSuite) Test_ReapProfiles_DefaultsRetention() {
	s.writeProfile("app-1-env-1", nixstore.DefaultRetentionDays*24*time.Hour+time.Hour)

	reaped, err := nixstore.ReapProfiles(nixstore.ReapProfilesParams{})

	s.NoError(err)
	s.Equal(1, reaped)
}

func (s *ProfileSuite) Test_PruneGenerations() {
	profile := filepath.Join(nixstore.ProfilesDir, "app-1-env-1")

	s.mockCommand.On("SetOpts", sys.CommandOpts{
		Name: "nix-env",
		Args: []string{"--profile", profile, "--delete-generations", "+1"},
	}).Return(s.mockCommand).Once()

	s.mockCommand.On("CombinedOutput").Return([]byte(""), nil).Once()

	s.NoError(nixstore.PruneGenerations(context.Background(), nixstore.PruneGenerationsParams{ProfilePath: profile}))
	s.mockCommand.AssertExpectations(s.T())
}

func (s *ProfileSuite) Test_PruneGenerations_WithoutProfile() {
	s.NoError(nixstore.PruneGenerations(context.Background(), nixstore.PruneGenerationsParams{}))
	s.mockCommand.AssertNotCalled(s.T(), "CombinedOutput")
}

// A failure has to name the profile and carry the command output: a bare exit
// status leaves an operator nothing to act on.
func (s *ProfileSuite) Test_PruneGenerations_ReportsOutput() {
	profile := filepath.Join(nixstore.ProfilesDir, "app-1-env-1")

	s.mockCommand.On("SetOpts", mock.Anything).Return(s.mockCommand).Once()
	s.mockCommand.On("CombinedOutput").Return([]byte("permission denied"), errCommandFailed).Once()

	err := nixstore.PruneGenerations(context.Background(), nixstore.PruneGenerationsParams{ProfilePath: profile})

	s.Error(err)
	s.Contains(err.Error(), "permission denied")
	s.Contains(err.Error(), profile)
}

func (s *ProfileSuite) Test_PruneAllGenerations() {
	first := s.writeProfile("app-1-env-1", time.Hour)
	second := s.writeProfile("app-2-env-2", time.Hour)

	for _, profile := range []string{second, first} {
		s.mockCommand.On("SetOpts", sys.CommandOpts{
			Name: "nix-env",
			Args: []string{"--profile", profile, "--delete-generations", "+1"},
		}).Return(s.mockCommand).Once()
	}

	s.mockCommand.On("CombinedOutput").Return([]byte(""), nil).Twice()

	s.NoError(nixstore.PruneAllGenerations(context.Background()))
	s.mockCommand.AssertExpectations(s.T())
}

func TestProfileSuite(t *testing.T) {
	suite.Run(t, new(ProfileSuite))
}
