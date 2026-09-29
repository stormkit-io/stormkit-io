package user_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stretchr/testify/suite"
)

type TokenPurposeSuite struct {
	suite.Suite
}

func (s *TokenPurposeSuite) issue(purpose user.Purpose, claims jwt.MapClaims) string {
	token, err := user.JWT(user.JWTParams{Purpose: purpose, Claims: claims})
	s.Require().NoError(err)

	return token
}

func (s *TokenPurposeSuite) parse(token string, purposes ...user.Purpose) jwt.MapClaims {
	return user.ParseJWT(&user.ParseJWTArgs{Bearer: token, Purposes: purposes})
}

func (s *TokenPurposeSuite) Test_AcceptedForItsPurpose() {
	token := s.issue(user.PurposeSession, jwt.MapClaims{"uid": "1"})

	claims := s.parse(token, user.PurposeSession)

	s.Require().NotNil(claims)
	s.Equal("1", claims["uid"])
	s.NotNil(s.parse(token, user.PurposeOAuthState, user.PurposeSession))
}

// Test_RejectedForOtherPurposes verifies that a token issued for one flow is
// not accepted by another, even with the same signing secret.
func (s *TokenPurposeSuite) Test_RejectedForOtherPurposes() {
	state := s.issue(user.PurposeOAuthState, jwt.MapClaims{"provider": "github", "uid": "1"})

	s.Nil(s.parse(state, user.PurposeSession))
	s.Equal(0, int(user.UIDFromBearer(state)))
}

// Test_ParseRequiresPurpose verifies that parsing fails closed when the caller
// does not say which purpose it expects.
func (s *TokenPurposeSuite) Test_ParseRequiresPurpose() {
	token := s.issue(user.PurposeSession, jwt.MapClaims{"uid": "1"})

	s.Nil(s.parse(token))
}

func (s *TokenPurposeSuite) Test_IssueRequiresPurpose() {
	_, err := user.JWT(user.JWTParams{Claims: jwt.MapClaims{"uid": "1"}})

	s.ErrorIs(err, user.ErrMissingPurpose)
}

// Test_ClaimsCannotOverridePurpose verifies that a caller-supplied purpose
// claim is ignored in favor of the declared purpose.
func (s *TokenPurposeSuite) Test_ClaimsCannotOverridePurpose() {
	token := s.issue(user.PurposeOAuthState, jwt.MapClaims{"purpose": string(user.PurposeSession), "uid": "1"})

	s.Nil(s.parse(token, user.PurposeSession))
	s.NotNil(s.parse(token, user.PurposeOAuthState))
}

// Test_LegacyTokenRejected verifies that tokens issued before purposes existed
// are no longer accepted.
func (s *TokenPurposeSuite) Test_LegacyTokenRejected() {
	legacy := jwt.New(jwt.GetSigningMethod("HS256"))
	legacy.Claims = jwt.MapClaims{"uid": "1", "issued": time.Now().Unix()}

	token, err := legacy.SignedString([]byte(config.AppSecret()))
	s.Require().NoError(err)

	s.Nil(s.parse(token, user.PurposeSession))
}

func TestTokenPurposeSuite(t *testing.T) {
	suite.Run(t, new(TokenPurposeSuite))
}
