package user_test

import (
	"encoding/json"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type UserModelSuite struct {
	suite.Suite
}

func (s *UserModelSuite) AfterTest(_, _ string) {
	admin.SetConfig(nil)
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

func (s *UserModelSuite) Test_IsAuthorizedToLogin_Waitlist() {
	cnf := &admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				SignUpMode: admin.SIGNUP_MODE_WAITLIST,
				Whitelist:  []string{"example.com", "stormkit.io"},
			},
		},
	}

	admin.SetConfig(cnf)

	expectations := map[string]bool{
		"test@example.com": true,
		"test@stormkit.io": true,
		"test@STORMKIT.io": true,
		"test@test.com":    false,
	}

	for email, expected := range expectations {
		u := user.New(email)
		s.Equal(expected, u.IsAuthorizedToLogin())
	}
}

// All users are authorized to login when sign up mode is "on".
func (s *UserModelSuite) Test_IsAuthorizedToLogin_On() {
	cnf := &admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				SignUpMode: admin.SIGNUP_MODE_ON,
			},
		},
	}

	admin.SetConfig(cnf)

	expectations := map[string]bool{
		"test@example.com": true,
		"test@stormkit.io": true,
		"test@STORMKIT.io": true,
		"test@test.com":    true,
	}

	for email, expected := range expectations {
		u := user.New(email)
		s.Equal(expected, u.IsAuthorizedToLogin())
	}
}

// No user is authorized to login when sign up mode is "off".
func (s *UserModelSuite) Test_IsAuthorizedToLogin_Off() {
	cnf := &admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				SignUpMode: admin.SIGNUP_MODE_OFF,
			},
		},
	}

	admin.SetConfig(cnf)

	expectations := map[string]bool{
		"test@example.com": false,
		"test@stormkit.io": false,
		"test@STORMKIT.io": false,
		"test@test.com":    false,
	}

	for email, expected := range expectations {
		u := user.New(email)
		s.Equal(expected, u.IsAuthorizedToLogin())
	}

	// Previously approved users are still authorized to login
	u := user.New("test@example.com")
	u.IsApproved = null.BoolFrom(true)
	s.True(u.IsAuthorizedToLogin())
}

func TestUser(t *testing.T) {
	suite.Run(t, &UserModelSuite{})
}
