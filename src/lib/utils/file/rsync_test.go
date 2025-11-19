//go:build !windows

package file_test

import (
	"errors"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type RsyncSuite struct {
	suite.Suite
	mockCmd *mocks.CommandInterface
}

func (s *RsyncSuite) BeforeTest(_, _ string) {
	s.mockCmd = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCmd
}

func (s *RsyncSuite) AfterTest(_, _ string) {
	sys.DefaultCommand = nil
}

func (s *RsyncSuite) Test_Rsync() {
	args := file.RsyncArgs{
		Source:      "/source/path",
		Destination: "/dest/path",
		WorkDir:     "/work/dir",
	}

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "rsync",
		Args: []string{"-a", "-R", args.Source, args.Destination},
		Dir:  args.WorkDir,
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil).Once()

	err := file.Rsync(args)
	s.NoError(err)

	s.mockCmd.AssertExpectations(s.T())
}

func (s *RsyncSuite) TestRsync_Error() {
	args := file.RsyncArgs{
		Source:      "/source/path",
		Destination: "/dest/path",
		WorkDir:     "/work/dir",
	}

	expectedErr := errors.New("rsync failed")

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "rsync",
		Args: []string{"-a", "-R", args.Source, args.Destination},
		Dir:  args.WorkDir,
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(expectedErr).Once()

	err := file.Rsync(args)
	s.Error(err)
	s.Equal(expectedErr, err)

	s.mockCmd.AssertExpectations(s.T())
}

func TestRsyncSuite(t *testing.T) {
	suite.Run(t, &RsyncSuite{})
}
