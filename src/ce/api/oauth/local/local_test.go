package local_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/local"
	"github.com/stretchr/testify/suite"
)

type LocalSuite struct {
	suite.Suite
}

func (s *LocalSuite) Test_IsLocal() {
	s.True(local.IsLocal("local/srv/repos/foo"))
	s.False(local.IsLocal("github/stormkit-io/foo"))
	s.False(local.IsLocal(""))
}

func (s *LocalSuite) Test_Path() {
	s.Equal("/srv/repos/foo", local.Path("local/srv/repos/foo"))
	s.Equal("", local.Path("github/stormkit-io/foo"))
}

func (s *LocalSuite) Test_CloneURL() {
	s.Equal("file:///srv/repos/foo", local.CloneURL("local/srv/repos/foo"))
	s.Equal("", local.CloneURL("github/stormkit-io/foo"))
}

func (s *LocalSuite) Test_FromURL() {
	s.Equal("local/srv/repos/foo", local.FromURL("file:///srv/repos/foo"))
	s.Equal("", local.FromURL("https://github.com/foo/bar"))
	s.Equal("", local.FromURL("file://"))
	s.Equal("", local.FromURL("file:///"))
}

func TestLocalSuite(t *testing.T) {
	suite.Run(t, new(LocalSuite))
}
