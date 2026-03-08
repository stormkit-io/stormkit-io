package publicapiv1_test

import (
	"testing"

	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stretchr/testify/suite"
)

type ValidatorsSuite struct {
	suite.Suite
	v *publicapiv1.Validators
}

func (s *ValidatorsSuite) SetupTest() {
	s.v = &publicapiv1.Validators{}
}

func (s *ValidatorsSuite) Test_ToInt_Empty_ReturnsZero() {
	i, err := s.v.ToInt("", "from")
	s.NoError(err)
	s.Equal(0, i)
}

func (s *ValidatorsSuite) Test_ToInt_ValidInteger() {
	i, err := s.v.ToInt("42", "from")
	s.NoError(err)
	s.Equal(42, i)
}

func (s *ValidatorsSuite) Test_ToInt_Zero() {
	i, err := s.v.ToInt("0", "from")
	s.NoError(err)
	s.Equal(0, i)
}

func (s *ValidatorsSuite) Test_ToInt_Negative_ReturnsError() {
	_, err := s.v.ToInt("-1", "from")
	s.EqualError(err, "The 'from' parameter cannot be smaller than 0")
}

func (s *ValidatorsSuite) Test_ToInt_NonInteger_ReturnsError() {
	_, err := s.v.ToInt("abc", "teamId")
	s.EqualError(err, "The 'teamId' parameter must be a valid integer")
}

func (s *ValidatorsSuite) Test_ToInt_Float_ReturnsError() {
	_, err := s.v.ToInt("1.5", "from")
	s.EqualError(err, "The 'from' parameter must be a valid integer")
}

func (s *ValidatorsSuite) Test_NormalizeRepo_Empty_IsValid() {
	normalized, ok := s.v.NormalizeRepo("")
	s.True(ok)
	s.Equal("", normalized)
}

func (s *ValidatorsSuite) Test_NormalizeRepo_Github() {
	normalized, ok := s.v.NormalizeRepo("github/org/repo")
	s.True(ok)
	s.Equal("github/org/repo", normalized)
}

func (s *ValidatorsSuite) Test_NormalizeRepo_Github_MixedCase() {
	normalized, ok := s.v.NormalizeRepo("GITHUB/Org/Repo")
	s.True(ok)
	s.Equal("github/org/repo", normalized)
}

func (s *ValidatorsSuite) Test_NormalizeRepo_Gitlab() {
	normalized, ok := s.v.NormalizeRepo("gitlab/org/repo")
	s.True(ok)
	s.Equal("gitlab/org/repo", normalized)
}

func (s *ValidatorsSuite) Test_NormalizeRepo_Bitbucket() {
	normalized, ok := s.v.NormalizeRepo("bitbucket/org/repo")
	s.True(ok)
	s.Equal("bitbucket/org/repo", normalized)
}

func (s *ValidatorsSuite) Test_NormalizeRepo_PrefixOnly_Invalid() {
	_, ok := s.v.NormalizeRepo("github/")
	s.False(ok)
}

func (s *ValidatorsSuite) Test_NormalizeRepo_MissingRepo_Invalid() {
	_, ok := s.v.NormalizeRepo("github/org")
	s.False(ok)
}

func (s *ValidatorsSuite) Test_NormalizeRepo_InvalidPrefix() {
	_, ok := s.v.NormalizeRepo("codeberg/org/repo")
	s.False(ok)
}

func (s *ValidatorsSuite) Test_NormalizeRepo_NoSlash() {
	_, ok := s.v.NormalizeRepo("myrepo")
	s.False(ok)
}

func TestValidators(t *testing.T) {
	suite.Run(t, new(ValidatorsSuite))
}
