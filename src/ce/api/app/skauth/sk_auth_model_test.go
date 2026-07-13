package skauth_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stretchr/testify/suite"
)

type UserModelSuite struct {
	suite.Suite
}

func (s *UserModelSuite) Test_UserMetadata_ValueScan_RoundTrip() {
	original := skauth.UserMetadata{
		Username:   "johndoe",
		ProfileURL: "https://x.com/johndoe",
	}

	value, err := original.Value()
	s.Require().NoError(err)

	// Value() must return a JSON string (not encrypted bytea) so Postgres can
	// store it as queryable JSONB.
	str, ok := value.(string)
	s.Require().True(ok, "Value() should return a string for the JSONB column")
	s.Equal(`{"username":"johndoe","profileUrl":"https://x.com/johndoe"}`, str)

	var scanned skauth.UserMetadata
	s.NoError(scanned.Scan([]byte(str)))
	s.Equal(original, scanned)

	// Scanning the string form works too.
	var scannedStr skauth.UserMetadata
	s.NoError(scannedStr.Scan(str))
	s.Equal(original, scannedStr)
}

func (s *UserModelSuite) Test_UserMetadata_Scan_NullAndEmpty() {
	var m skauth.UserMetadata

	s.NoError(m.Scan(nil), "NULL leaves the zero struct")
	s.Equal(skauth.UserMetadata{}, m)

	s.NoError(m.Scan([]byte(`{}`)), "empty object leaves the zero struct")
	s.Equal(skauth.UserMetadata{}, m)

	s.NoError(m.Scan([]byte(``)), "empty bytes are a no-op")
	s.Equal(skauth.UserMetadata{}, m)
}

func (s *UserModelSuite) Test_User_JSON_IncludesMetadataOmitsSecrets() {
	u := &skauth.User{
		UUID:         "user-uuid",
		FirstName:    "John",
		LastName:     "Doe",
		Email:        "john@example.com",
		Avatar:       "https://example.com/a.jpg",
		PasswordHash: "should-not-leak",
		Metadata: skauth.UserMetadata{
			Username:   "johndoe",
			ProfileURL: "https://x.com/johndoe",
		},
	}

	j := u.JSON()

	s.Equal("user-uuid", j["id"])
	s.Equal("john@example.com", j["email"])
	s.Equal("johndoe", j["username"])
	s.Equal("https://x.com/johndoe", j["profileUrl"])

	_, hasHash := j["passwordHash"]
	s.False(hasHash, "password hash must never be serialised")

	_, hasInternalID := j["ID"]
	s.False(hasInternalID, "internal numeric id must never be serialised")
}

func TestUserModelSuite(t *testing.T) {
	suite.Run(t, new(UserModelSuite))
}
