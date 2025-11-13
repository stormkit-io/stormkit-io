package factory

import (
	"fmt"
	"time"

	"github.com/gosimple/slug"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

var userCounter = 0

type MockUser struct {
	*user.User
	*Factory
	DefaultTeamID types.ID
}

func (u *MockUser) Insert(conn databasetest.TestDB) error {
	if !u.CreatedAt.Valid {
		t, err := time.Parse(time.DateTime, "2019-02-26 15:00:00")

		if err != nil {
			return err
		}

		u.CreatedAt = utils.Unix{
			Time:  t,
			Valid: true,
		}
	}

	return conn.PrepareOrPanic(`
		WITH
			new_user AS (
				INSERT INTO users (
					first_name, last_name, display_name,
					avatar_uri, is_admin, created_at, is_approved, metadata
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8
				)
				RETURNING user_id
			),
			new_team AS (
				INSERT INTO teams
					(team_name, team_slug, user_id, is_default, created_at)
				VALUES
					($9, $10, (SELECT user_id FROM new_user), TRUE, NOW())
				RETURNING team_id
			),
			new_team_member AS (
				INSERT INTO team_members
					(team_id, user_id, member_role, membership_status)
				SELECT
					(SELECT team_id FROM new_team),
					(SELECT user_id FROM new_user),
					'owner',
					TRUE
			)
			INSERT INTO user_emails
				(user_id, email, is_verified, is_primary)
			SELECT
				(SELECT user_id FROM new_user),
				$11,
				TRUE,
				TRUE
			RETURNING
				(SELECT user_id FROM new_user),
				(SELECT team_id FROM new_team)
			`,
	).QueryRow(
		u.FirstName, u.LastName, u.DisplayName,
		u.Avatar, u.IsAdmin, u.CreatedAt, u.IsApproved, u.Metadata,
		team.DEFAULT_TEAM_NAME, slug.Make(team.DEFAULT_TEAM_NAME),
		u.PrimaryEmail(),
	).Scan(&u.ID, &u.DefaultTeamID)
}

// GetUser returns the first application that was created
// in this factory. If none is found, it will create a new one.
func (f *Factory) GetUser() *MockUser {
	res := factoryLookup[MockUser](f)
	if res != nil {
		return res
	}

	return f.MockUser()
}

func (f *Factory) SeedProviders(usr *MockUser) {
	providers := map[string]string{
		"github":    fmt.Sprintf("https://api.github.com/users/%s", usr.DisplayName),
		"gitlab":    fmt.Sprintf("https://gitlab.com/%s", usr.DisplayName),
		"bitbucket": fmt.Sprintf("https://bitbucket.org/%s", usr.DisplayName),
	}

	for provider, accountUri := range providers {
		accessToken := null.NewString("4592-vxay", provider == "gitlab")

		_, err := f.conn.PrepareOrPanic(`
			INSERT INTO user_access_tokens (
				user_id, display_name, account_uri, provider,
				token_type, token_value, token_refresh, personal_access_token,
				expire_at
			) VALUES (
				$1, $2, $3, $4,
				$5, $6, $7, $8, '2050-02-02'::timestamp
			)`,
		).Exec(
			usr.ID, usr.DisplayName, accountUri, provider,
			"bearer", "1234-abcd-4251", "6431-refresh", accessToken,
		)

		if err != nil {
			panic(err)
		}
	}
}

func (f *Factory) MockUser(overwrites ...map[string]interface{}) *MockUser {
	userCounter = userCounter + 1
	usr := user.New(fmt.Sprintf("test-%d@stormkit.io", userCounter))
	usr.Avatar = null.NewString("https://avatars3.githubusercontent.com/u/55663230?v=4", true)
	usr.FirstName = null.NewString("David", true)
	usr.LastName = null.NewString("Lorenzo", true)
	usr.DisplayName = "dlorenzo"
	usr.IsAdmin = false
	usr.IsApproved = null.BoolFrom(true)
	usr.Metadata = user.UserMeta{
		SeatsPurchased: 1,
		PackageName:    config.PackagePremium,
	}

	for _, o := range overwrites {
		merge(usr, o)
	}

	mock := f.newObject(&MockUser{
		User:    usr,
		Factory: f,
	}).(*MockUser)

	if err := mock.Insert(f.conn); err != nil {
		panic(err)
	}

	f.SeedProviders(mock)

	return mock
}
