//go:build windows

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

func (s *RsyncSuite) TestRsync_File() {
	args := file.RsyncArgs{
		Source:      "C:\\source\\path\\file.txt",
		Destination: "C:\\dest\\path",
		WorkDir:     "C:\\work\\dir",
	}

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "robocopy",
		Args: []string{"C:\\source\\path", args.Destination, "file.txt", "/E", "/DCOPY:DAT", "/R:0", "/W:0"},
		Dir:  args.WorkDir,
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil).Once()

	err := file.Rsync(args)
	s.NoError(err)

	s.mockCmd.AssertExpectations(s.T())
}

func (s *RsyncSuite) TestRsync_Directory() {
	args := file.RsyncArgs{
		Source:      "C:\\source\\path",
		Destination: "C:\\dest\\path",
		WorkDir:     "C:\\work\\dir",
	}

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "robocopy",
		Args: []string{args.Source, args.Destination, "*.*", "/E", "/DCOPY:DAT", "/R:0", "/W:0"},
		Dir:  args.WorkDir,
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil).Once()

	err := file.Rsync(args)
	s.NoError(err)

	s.mockCmd.AssertExpectations(s.T())
}

func (s *RsyncSuite) TestRsync_Error() {
	args := file.RsyncArgs{
		Source:      "C:\\source\\path",
		Destination: "C:\\dest\\path",
		WorkDir:     "C:\\work\\dir",
	}

	expectedErr := errors.New("robocopy failed")

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "robocopy",
		Args: []string{args.Source, args.Destination, "*.*", "/E", "/DCOPY:DAT", "/R:0", "/W:0"},
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
