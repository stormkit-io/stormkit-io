package buildconf_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stretchr/testify/suite"
)

type WatchPathsSuite struct {
	suite.Suite
}

func (s *WatchPathsSuite) Test_ValidPaths() {
	s.Empty(buildconf.ValidateWatchPaths(nil))
	s.Empty(buildconf.ValidateWatchPaths([]string{}))
	s.Empty(buildconf.ValidateWatchPaths([]string{"packages/ui", "/packages/config/", "./shared", "a/b/c"}))
}

func (s *WatchPathsSuite) Test_InvalidPaths() {
	for _, p := range []string{"", " ", "~/home", "..", "../escape", "a/../../b"} {
		s.NotEmpty(buildconf.ValidateWatchPaths([]string{p}), "expected %q to be invalid", p)
	}
}

func (s *WatchPathsSuite) Test_MixedPaths_ReportsOnlyInvalid() {
	errs := buildconf.ValidateWatchPaths([]string{"packages/ui", "../escape"})

	s.Len(errs, 1)
	s.Contains(errs[0], "../escape")
}

func (s *WatchPathsSuite) Test_Normalize_TrimsAndDropsEmpty() {
	s.Equal(
		[]string{"packages/ui", "packages/config"},
		buildconf.NormalizeWatchPaths([]string{" packages/ui ", "", "  ", "packages/config"}),
	)
}

func TestWatchPathsSuite(t *testing.T) {
	suite.Run(t, new(WatchPathsSuite))
}
