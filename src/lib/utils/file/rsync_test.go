package file_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type RsyncSuite struct {
	suite.Suite
	mockCmd *mocks.CommandInterface
	tmpdir  string
}

func (s *RsyncSuite) BeforeTest(_, _ string) {
	var err error
	s.mockCmd = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCmd
	s.tmpdir, err = os.MkdirTemp("", "rsync_test")
	s.NoError(err)
}

func (s *RsyncSuite) AfterTest(_, _ string) {
	sys.DefaultCommand = nil
	os.RemoveAll(s.tmpdir)
}

func (s *RsyncSuite) TestRsync_Unix() {
	args := file.RsyncArgs{
		Context:     context.Background(),
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

func (s *RsyncSuite) TestRsync_Unix_Error() {
	args := file.RsyncArgs{
		Context:     context.Background(),
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

func (s *RsyncSuite) TestRsync_Windows() {
	config.IsWindows = true
	defer func() { config.IsWindows = false }()

	os.WriteFile(filepath.Join(s.tmpdir, "file.txt"), []byte("test"), 0644)

	args := file.RsyncArgs{
		Context:     context.Background(),
		Source:      "file.txt",
		Destination: "C:\\dest\\path",
		WorkDir:     s.tmpdir,
	}

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "robocopy",
		Args: []string{s.tmpdir, filepath.Join(s.tmpdir, "C:\\dest\\path"), "file.txt", "/E", "/DCOPY:DAT", "/R:0", "/W:0"},
		Dir:  args.WorkDir,
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil).Once()

	s.NoError(file.Rsync(args))

	s.mockCmd.AssertExpectations(s.T())
}

func (s *RsyncSuite) TestRsync_Windows_Error() {
	config.IsWindows = true
	defer func() { config.IsWindows = false }()

	args := file.RsyncArgs{
		Context:     context.Background(),
		Source:      "C:\\source\\path",
		Destination: "C:\\dest\\path",
		WorkDir:     s.tmpdir,
	}

	// This will fail because the source file does not exist
	s.Error(file.Rsync(args))
}

func TestRsyncSuite(t *testing.T) {
	suite.Run(t, &RsyncSuite{})
}
