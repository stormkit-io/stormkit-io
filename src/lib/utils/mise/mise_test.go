package mise_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type MiseSuite struct {
	suite.Suite
	mockCommand  *mocks.CommandInterface
	originalPath string
}

func (s *MiseSuite) SetupSuite() {
	s.originalPath = os.Getenv("PATH")
}

func (s *MiseSuite) BeforeTest(_, _ string) {
	mise.ResetCache()
	s.mockCommand = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCommand
}

func (s *MiseSuite) AfterTest(_, _ string) {
	sys.DefaultCommand = nil
	os.Setenv("PATH", s.originalPath)
}

func (s *MiseSuite) Test_InstallGlobal() {
	ctx := context.Background()

	s.mockCommand.On("SetOpts", sys.CommandOpts{Name: "mise", Args: []string{"use", "--global", "go@1.24"}}).Return(s.mockCommand).Once()
	s.mockCommand.On("CombinedOutput").Return([]byte("go@1.24"), nil).Once()

	s.mockCommand.On("SetOpts", sys.CommandOpts{Name: "mise", Args: []string{"bin-paths"}}).Return(s.mockCommand).Once()
	s.mockCommand.On("Output").Return([]byte("go@1.24/bin"), nil).Once()

	output, err := mise.Client().InstallGlobal(ctx, "go@1.24")
	s.NoError(err)
	s.Equal("go@1.24", output)

	// Should also update the path
	s.Equal(fmt.Sprintf("go@1.24/bin:%s", s.originalPath), os.Getenv("PATH"))
}

func TestMiseSuite(t *testing.T) {
	suite.Run(t, new(MiseSuite))
}
