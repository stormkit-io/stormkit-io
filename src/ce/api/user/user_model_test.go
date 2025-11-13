package user_test

import (
	"encoding/json"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type UserModelSuite struct {
	suite.Suite
}

func (s *UserModelSuite) Test_JSON() {
	u := user.New("test@example.org")
	u.FirstName = null.StringFrom("Jane")
	u.LastName = null.StringFrom("Doe")
	u.Avatar = null.StringFrom("https://example.org/avatars/jane")
	u.IsApproved = null.BoolFrom(false)
	u.IsAdmin = false
	u.DisplayName = "jane-example"

	expected := `{
		"avatar": "https://example.org/avatars/jane",
		"displayName": "jane-example",
		"email": "test@example.org",
		"fullName": "Jane Doe",
		"id": "0",
		"memberSince": 0,
		"package": {
			"id": "free",
			"seats": 1
		}
	}`

	b, err := json.Marshal(u.JSON())
	s.NoError(err)
	s.JSONEq(expected, string(b))
}

func TestUser(t *testing.T) {
	suite.Run(t, &UserModelSuite{})
}
