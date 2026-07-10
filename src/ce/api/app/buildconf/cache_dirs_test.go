package buildconf_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stretchr/testify/suite"
)

type ValidateCacheDirsSuite struct {
	suite.Suite
}

func (s *ValidateCacheDirsSuite) Test_ValidDirs() {
	s.Empty(buildconf.ValidateCacheDirs(nil))
	s.Empty(buildconf.ValidateCacheDirs([]string{}))
	s.Empty(buildconf.ValidateCacheDirs([]string{"node_modules", ".next/cache", "./dist", "a/b/c"}))
}

func (s *ValidateCacheDirsSuite) Test_InvalidDirs() {
	for _, dir := range []string{"", " ", "/absolute", "~/home", "..", "../escape", "a/../../b", "."} {
		s.NotEmpty(buildconf.ValidateCacheDirs([]string{dir}), "expected %q to be invalid", dir)
	}
}

func (s *ValidateCacheDirsSuite) Test_MixedDirs_ReportsOnlyInvalid() {
	errs := buildconf.ValidateCacheDirs([]string{"node_modules", "../escape"})

	s.Len(errs, 1)
	s.Contains(errs[0], "../escape")
}

func TestValidateCacheDirsSuite(t *testing.T) {
	suite.Run(t, new(ValidateCacheDirsSuite))
}
