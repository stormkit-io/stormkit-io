package nixstore_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
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
	s.restore = nixstore.SetLookPath(true)
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
	s.restore = nixstore.SetLookPath(false)

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

	s.mockCommand.On("Run").Return(nil).Once()

	s.NoError(nixstore.CollectGarbage(context.Background(), nixstore.CollectGarbageParams{RetentionDays: 3}))
	s.mockCommand.AssertExpectations(s.T())
}

func (s *NixStoreSuite) Test_CollectGarbage_DefaultsRetention() {
	s.mockCommand.On("SetOpts", sys.CommandOpts{
		Name: "nix-collect-garbage",
		Args: []string{"--delete-older-than", "7d"},
	}).Return(s.mockCommand).Once()

	s.mockCommand.On("Run").Return(nil).Once()

	s.NoError(nixstore.CollectGarbage(context.Background(), nixstore.CollectGarbageParams{}))
	s.mockCommand.AssertExpectations(s.T())
}

// Containers without Nix must not report an error: the same code path runs in
// images that never install it.
func (s *NixStoreSuite) Test_CollectGarbage_WithoutNix() {
	s.restore()
	s.restore = nixstore.SetLookPath(false)

	s.NoError(nixstore.CollectGarbage(context.Background(), nixstore.CollectGarbageParams{}))
	s.mockCommand.AssertNotCalled(s.T(), "Run")
}

func TestNixStoreSuite(t *testing.T) {
	suite.Run(t, new(NixStoreSuite))
}
