package utils_test

import (
	"regexp"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HashSuite struct {
	suite.Suite
}

func (s *HashSuite) Test_RandomToken_Format() {
	for _, n := range []int{1, 16, 48, 128} {
		token := utils.RandomToken(n)

		s.Len(token, n)
		s.Regexp(regexp.MustCompile(`^[a-zA-Z0-9]+$`), token)
	}
}

// Test_RandomToken_Unique verifies that tokens generated at the same moment
// differ, which a time-seeded generator does not guarantee.
func (s *HashSuite) Test_RandomToken_Unique() {
	seen := map[string]bool{}

	for range 1000 {
		token := utils.RandomToken(32)

		s.False(seen[token])
		seen[token] = true
	}
}

func TestHashSuite(t *testing.T) {
	suite.Run(t, new(HashSuite))
}
