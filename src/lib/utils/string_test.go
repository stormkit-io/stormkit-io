package utils_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type StringSuite struct {
	suite.Suite
}

func (s *StringSuite) Test_GetString() {
	s.Equal("c", utils.GetString("", "", "c", "d"))
	s.Equal("a", utils.GetString("a", "", "c", "d"))
}

func (s *StringSuite) Test_TrimPath() {
	s.Equal("/path/to/resource", utils.TrimPath("/path/to/resource/"))
	s.Equal("/path/to/resource", utils.TrimPath("path/to/resource/"))
	s.Equal("/path/to/resource", utils.TrimPath("/path/to/resource"))
	s.Equal("/path/to/resource", utils.TrimPath("path/to/resource"))
	s.Equal("/", utils.TrimPath("/"))
	s.Equal("", utils.TrimPath(""))
	s.Equal("", utils.TrimPath("   "))
}

func (s *StringSuite) Test_ParseSemver() {
	expections := map[string][3]string{
		"1.2.3":       {"1", "2", "3"},
		"0.0.1":       {"0", "0", "1"},
		"10.20.30":    {"10", "20", "30"},
		"v1.2.3":      {"1", "2", "3"},
		"v0.0.1":      {"0", "0", "1"},
		"v10.20.30":   {"10", "20", "30"},
		"1.2.3-alpha": {"1", "2", "3-alpha"},
		"v1.2.3-beta": {"1", "2", "3-beta"},
		"1.2":         {"1", "2", "0"},
		"v1.2":        {"1", "2", "0"},
		"1":           {"1", "0", "0"},
		"v1":          {"1", "0", "0"},
	}

	for input, expected := range expections {
		major, minor, patch := utils.ParseSemver(input)
		s.Equal(expected[0], major, "Major version mismatch for input: %s", input)
		s.Equal(expected[1], minor, "Minor version mismatch for input: %s", input)
		s.Equal(expected[2], patch, "Patch version mismatch for input: %s", input)
	}
}

func (s *StringSuite) Test_ParseRepoWithProvider() {
	type testCase struct {
		input    string
		provider string
		slug     string
	}

	cases := []testCase{
		{"https://github.com/my-org/my-repo", "github", "my-org/my-repo"},
		{"https://github.com/my-org/my-repo.git", "github", "my-org/my-repo"},
		{"http://github.com/my-org/my-repo", "github", "my-org/my-repo"},
		{"https://gitlab.com/my-org/my-repo", "gitlab", "my-org/my-repo"},
		{"https://gitlab.com/my-group/my-subgroup/my-repo", "gitlab", "my-group/my-subgroup/my-repo"},
		{"https://gitlab.com/my-group/my-subgroup/my-repo.git", "gitlab", "my-group/my-subgroup/my-repo"},
		{"gitlab/my-group/my-subgroup/my-repo", "gitlab", "my-group/my-subgroup/my-repo"},
		{"https://bitbucket.org/my-org/my-repo", "bitbucket", "my-org/my-repo"},
		{"github/my-org/my-repo", "github", "my-org/my-repo"},
		{"gitlab/my-org/my-repo", "gitlab", "my-org/my-repo"},
		{"bitbucket/my-org/my-repo", "bitbucket", "my-org/my-repo"},
		{"my-org/my-repo", "github", "my-org/my-repo"},
	}

	for _, tc := range cases {
		provider, slug := utils.ParseRepoWithProvider(tc.input)
		s.Equal(tc.provider, provider, "provider mismatch for input: %s", tc.input)
		s.Equal(tc.slug, slug, "slug mismatch for input: %s", tc.input)
	}
}

func TestString(t *testing.T) {
	suite.Run(t, &StringSuite{})
}
