package admin_test

import (
	"context"
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
	conn         databasetest.TestDB
	mockCommand  *mocks.CommandInterface
	originalPath string
	restore      func()
}

func (s *AdminMaintenanceSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	utils.SetAppKey([]byte(utils.RandomToken(32)))
	admin.ResetCache(context.Background())

	s.mockCommand = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCommand

	s.originalPath = nixstore.DefaultPath
	nixstore.DefaultPath = s.T().TempDir()
	s.restore = nixstoretest.StubLookPath(true)
}

func (s *AdminMaintenanceSuite) AfterTest(_, _ string) {
	sys.DefaultCommand = nil
	nixstore.DefaultPath = s.originalPath
	s.restore()
	admin.ResetCache(context.Background())
	s.conn.CloseTx()
}

func (s *AdminMaintenanceSuite) Test_NixRetentionDays_DefaultsWhenUnset() {
	s.Equal(nixstore.DefaultRetentionDays, admin.NixRetentionDays(context.Background()))
}

func (s *AdminMaintenanceSuite) Test_NixRetentionDays_UsesConfiguredValue() {
	ctx := context.Background()

	s.NoError(admin.Store().UpsertConfig(ctx, admin.InstanceConfig{
		SystemConfig: &admin.SystemConfig{NixRetentionDays: 21},
	}))

	admin.ResetCache(ctx)

	s.Equal(21, admin.NixRetentionDays(ctx))
}

func (s *AdminMaintenanceSuite) Test_CollectNixGarbage() {
	s.mockCommand.On("SetOpts", sys.CommandOpts{
		Name: "nix-collect-garbage",
		Args: []string{"--delete-older-than", "7d"},
	}).Return(s.mockCommand).Once()

	s.mockCommand.On("Run").Return(nil).Once()

	admin.CollectNixGarbage(context.Background())
	s.mockCommand.AssertExpectations(s.T())
}

// A container without Nix runs the same code path and must not shell out.
func (s *AdminMaintenanceSuite) Test_CollectNixGarbage_WithoutNix() {
	s.restore()
	s.restore = nixstoretest.StubLookPath(false)

	admin.CollectNixGarbage(context.Background())
	s.mockCommand.AssertNotCalled(s.T(), "Run")
}

// A failing collection is logged, not propagated: a full disk must not take
// the container down.
func (s *AdminMaintenanceSuite) Test_CollectNixGarbage_SurvivesFailure() {
	s.mockCommand.On("SetOpts", sys.CommandOpts{
		Name: "nix-collect-garbage",
		Args: []string{"--delete-older-than", "7d"},
	}).Return(s.mockCommand).Once()

	s.mockCommand.On("Run").Return(assertError{}).Once()

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
