package user_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/suite"
)

type UserStoreSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *UserStoreSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *UserStoreSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *UserStoreSuite) Test_MustUser() {
	store := user.NewStore()
	ctx := context.Background()
	user, err := store.MustUser(&oauth.User{
		Emails: []oauth.Email{{Address: "test@stormkit.io", IsPrimary: true, IsVerified: true}},
	})

	s.NoError(err)
	s.NotNil(user)
	s.Greater(int64(user.ID), int64(0))
	s.Equal("test@stormkit.io", user.PrimaryEmail())
	s.False(user.IsAdmin)
	s.True(user.IsApproved.ValueOrZero())

	teams, err := team.NewStore().Teams(ctx, user.ID)
	s.NoError(err)
	s.Len(teams, 1)
	s.Equal("owner", teams[0].CurrentUserRole)
	s.Equal(team.DEFAULT_TEAM_NAME, teams[0].Name)

	apiKeys, err := apikey.NewStore().APIKeys(ctx, user.ID, apikey.SCOPE_USER)
	s.NoError(err)
	s.Len(apiKeys, 1)
}

func (s *UserStoreSuite) Test_MustUser_Approval() {
	config.SetIsSelfHosted(true)
	defer config.SetIsSelfHosted(false)

	cfg := admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				SignUpMode: admin.SIGNUP_MODE_WAITLIST,
			},
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cfg))

	user, err := user.NewStore().MustUser(&oauth.User{
		Emails: []oauth.Email{{Address: "test@stormkit.io", IsPrimary: true, IsVerified: true}},
	})

	s.NoError(err)
	s.False(user.IsApproved.Valid)
}

func (s *UserStoreSuite) Test_MustUser_Approval_Whitelisted() {
	config.SetIsSelfHosted(true)
	defer config.SetIsSelfHosted(false)

	cfg := admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				SignUpMode: admin.SIGNUP_MODE_WAITLIST,
				Whitelist:  []string{"stormkit.io"},
			},
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cfg))

	user, err := user.NewStore().MustUser(&oauth.User{
		Emails: []oauth.Email{{Address: "test@STORMKIT.io", IsPrimary: true, IsVerified: true}},
	})

	s.NoError(err)
	s.True(user.IsApproved.Valid)
	s.True(user.IsApproved.ValueOrZero())
}

func (s *UserStoreSuite) Test_MustUser_IsAdmin() {
	store := user.NewStore()
	ctx := context.Background()
	user, err := store.MustUser(&oauth.User{
		Emails:  []oauth.Email{{Address: "test@stormkit.io", IsPrimary: true, IsVerified: true}},
		IsAdmin: true,
	})

	s.NoError(err)
	s.NotNil(user)
	s.Greater(int64(user.ID), int64(0))
	s.Equal("test@stormkit.io", user.PrimaryEmail())
	s.True(user.IsAdmin)

	teams, err := team.NewStore().Teams(ctx, user.ID)
	s.NoError(err)
	s.Len(teams, 1)
	s.Equal("owner", teams[0].CurrentUserRole)
	s.Equal(team.DEFAULT_TEAM_NAME, teams[0].Name)

	apiKeys, err := apikey.NewStore().APIKeys(ctx, user.ID, apikey.SCOPE_USER)
	s.NoError(err)
	s.Len(apiKeys, 1)
}

func (s *UserStoreSuite) Test_InsertEmails() {
	usr := s.MockUser()
	store := user.NewStore()
	emails := []oauth.Email{
		{Address: "test@stormkit.io", IsPrimary: true, IsVerified: true},
		{Address: "hello@stormkit.io", IsPrimary: false, IsVerified: true},
	}

	s.NoError(store.InsertEmails(context.Background(), usr.ID, emails))

	updatedUser, err := store.UserByID(usr.ID)

	s.NoError(err)
	s.Equal([]oauth.Email{
		usr.Emails[0],
		{Address: "test@stormkit.io", IsPrimary: true, IsVerified: true},
		{Address: "hello@stormkit.io", IsPrimary: false, IsVerified: true},
	}, updatedUser.Emails)
}

func (s *UserStoreSuite) Test_UpdateSubscription() {
	usr := s.MockUser()
	ctx := context.Background()
	store := user.NewStore()

	err := store.UpdateSubscription(ctx, usr.ID, user.UserMeta{
		PackageName:      config.PackageUltimate,
		SeatsPurchased:   5,
		StripeCustomerID: "cus_test123",
	})

	s.NoError(err)

	updatedUser, err := store.UserByID(usr.ID)

	s.NoError(err)
	s.Equal(config.PackageUltimate, updatedUser.Metadata.PackageName)

	// Now let's generate a license for self-hosted
	license, err := store.GenerateSelfHostedLicense(ctx, 5, usr.ID, nil)
	s.NoError(err)
	s.NotNil(license)

	license, err = store.LicenseByUserID(ctx, license.UserID)
	s.NoError(err)
	s.NotNil(license)
	s.Equal(5, license.Seats)

	var metadata []byte

	expected := `{
		"stripeCustomerId": "cus_test123",
		"package": "ultimate",
		"seats": 5
	}`

	s.NoError(s.conn.QueryRow("SELECT metadata FROM skitapi.users WHERE user_id = $1", usr.ID).Scan(&metadata))
	s.JSONEq(expected, string(metadata))
}

func TestUserStore(t *testing.T) {
	suite.Run(t, &UserStoreSuite{})
}
