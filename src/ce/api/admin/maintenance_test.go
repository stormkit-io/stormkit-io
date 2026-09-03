package admin_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore/nixstoretest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type AdminMaintenanceSuite struct {
	suite.Suite
	conn             databasetest.TestDB
	mockCommand      *mocks.CommandInterface
	originalPath     string
	originalProfiles string
	restore          func()
}

func (s *AdminMaintenanceSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	utils.SetAppKey([]byte(utils.RandomToken(32)))
	admin.ResetCache(context.Background())

	s.mockCommand = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCommand

	s.originalPath = nixstore.DefaultPath
	s.originalProfiles = nixstore.ProfilesDir
	nixstore.DefaultPath = s.T().TempDir()
	nixstore.ProfilesDir = filepath.Join(s.T().TempDir(), "stormkit")
	s.restore = nixstoretest.StubLookPath(true)
}

func (s *AdminMaintenanceSuite) AfterTest(_, _ string) {
	sys.DefaultCommand = nil
	nixstore.DefaultPath = s.originalPath
	nixstore.ProfilesDir = s.originalProfiles
	s.restore()
	admin.ResetCache(context.Background())
	s.conn.CloseTx()
}

func (s *AdminMaintenanceSuite) Test_CollectNixGarbage() {
	s.mockCommand.On("SetOpts", sys.CommandOpts{
		Name: "nix-collect-garbage",
		Args: []string{"--delete-older-than", "7d"},
	}).Return(s.mockCommand).Once()

	s.mockCommand.On("CombinedOutput").Return([]byte(""), nil).Once()

	admin.CollectNixGarbage(context.Background())
	s.mockCommand.AssertExpectations(s.T())
}

// A container without Nix runs the same code path and must not shell out.
func (s *AdminMaintenanceSuite) Test_CollectNixGarbage_WithoutNix() {
	s.restore()
	s.restore = nixstoretest.StubLookPath(false)

	admin.CollectNixGarbage(context.Background())
	s.mockCommand.AssertNotCalled(s.T(), "CombinedOutput")
}

// A failing collection is logged, not propagated: a full disk must not take
// the container down.
func (s *AdminMaintenanceSuite) Test_CollectNixGarbage_SurvivesFailure() {
	s.mockCommand.On("SetOpts", sys.CommandOpts{
		Name: "nix-collect-garbage",
		Args: []string{"--delete-older-than", "7d"},
	}).Return(s.mockCommand).Once()

	s.mockCommand.On("CombinedOutput").Return([]byte("cannot open lock file"), assertError{}).Once()

	s.NotPanics(func() { admin.CollectNixGarbage(context.Background()) })
}

func (s *AdminMaintenanceSuite) Test_StartDiskMaintenance_ReturnsWithoutNix() {
	s.restore()
	s.restore = nixstoretest.StubLookPath(false)

	done := make(chan struct{})

	go func() {
		admin.StartDiskMaintenance(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		s.Fail("StartDiskMaintenance did not return on a container without Nix")
	}
}

type assertError struct{}

func (assertError) Error() string { return "boom" }

func TestAdminMaintenanceSuite(t *testing.T) {
	suite.Run(t, new(AdminMaintenanceSuite))
}
