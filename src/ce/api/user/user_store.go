package user

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gosimple/slug"
	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/discord"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	null "gopkg.in/guregu/null.v3"
)

// Store handles user logic in the database.
type Store struct {
	*database.Store
	selectTmpl            *template.Template
	insertTmpl            *template.Template
	insertEmailsTmpl      *template.Template
	selectLicenseTmpl     *template.Template
	userMetricsTmpl       *template.Template
	updateUserMetricsTmpl *template.Template
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{
		Store:             database.NewStore(),
		userMetricsTmpl:   template.Must(template.New("userMetrics").Parse(ustmt.userMetrics)),
		selectTmpl:        template.Must(template.New("selectUsers").Parse(ustmt.selectUsers)),
		selectLicenseTmpl: template.Must(template.New("selectLicense").Parse(ustmt.selectLicense)),
		updateUserMetricsTmpl: template.Must(
			template.New("updateUserMetrics").
				Funcs(template.FuncMap{"generateValues": utils.GenerateValues}).
				Parse(ustmt.updateUsageMetrics)),
		insertTmpl: template.Must(
			template.New("insertUser").
				Funcs(template.FuncMap{
					"last": func(x int, a any) bool {
						return x == reflect.ValueOf(a).Len()-1
					},
				}).
				Parse(ustmt.insertUser)),
		insertEmailsTmpl: template.Must(
			template.New("insertEmails").
				Funcs(template.FuncMap{
					"generateValues": utils.GenerateValues,
				}).
				Parse(ustmt.insertEmails)),
	}
}

func (s *Store) selectUsers(ctx context.Context, query string, params ...any) ([]*User, error) {
	var emails []byte

	rows, err := s.Query(ctx, query, params...)

	if rows == nil || err == sql.ErrNoRows {
		return nil, err
	}

	defer rows.Close()

	users := []*User{}

	for rows.Next() {
		user := &User{}

		err := rows.Scan(
			&user.ID, &user.Avatar, &user.DisplayName,
			&user.FirstName, &user.LastName, &emails,
			&user.CreatedAt, &user.IsAdmin,
			&user.Metadata, &user.LastLogin, &user.IsApproved,
		)

		if err != nil {
			return nil, err
		}

		if emails != nil {
			if err := json.Unmarshal(emails, &user.Emails); err != nil {
				return nil, err
			}
		}

		if user.Metadata.PackageName == "" {
			user.Metadata.PackageName = config.PackageFree
		}

		if user.Metadata.SeatsPurchased == 0 {
			user.Metadata.SeatsPurchased = 1
		}

		users = append(users, user)
	}

	return users, err
}

// UserByID queries a user by its user id.
func (s *Store) UserByID(userID types.ID) (*User, error) {
	var wr bytes.Buffer

	data := map[string]any{
		"where": "u.user_id = $1 AND u.deleted_at IS NULL",
		"limit": 1,
	}

	if err := s.selectTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	users, err := s.selectUsers(context.TODO(), wr.String(), userID)

	if err != nil || len(users) == 0 {
		return nil, err
	}

	return users[0], nil
}

// PendingUsers returns the list of users pending approval.
func (s *Store) PendingUsers(ctx context.Context) ([]*User, error) {
	var wr bytes.Buffer

	data := map[string]any{
		"where": "u.deleted_at IS NULL AND u.is_approved IS NULL",
		"limit": 100,
	}

	if err := s.selectTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	return s.selectUsers(ctx, wr.String())
}

// UpdateApprovalStatus updates the approval status for the given user ids.
func (s *Store) UpdateApprovalStatus(ctx context.Context, userIDs []types.ID, approved bool) error {
	_, err := s.Exec(ctx, ustmt.updateApprovalStatus, approved, pq.Array(userIDs))
	return err
}

// TeamOwner returns the owner user of a team.
func (s *Store) TeamOwner(teamID types.ID) (*User, error) {
	var wr bytes.Buffer

	data := map[string]any{
		"where": `u.deleted_at IS NULL AND u.user_id IN (
			SELECT
				tm.user_id
			FROM
				team_members tm
			WHERE
				tm.member_role = 'owner' AND
				tm.team_id = $1
		)`,
		"limit": 1,
	}

	if err := s.selectTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	users, err := s.selectUsers(context.TODO(), wr.String(), teamID)

	if err != nil || len(users) == 0 {
		return nil, err
	}

	return users[0], nil
}

// UserByEmail queries a user by its email address.
func (s *Store) UserByEmail(ctx context.Context, emails []string) (*User, error) {
	var wr bytes.Buffer

	data := map[string]any{
		"where": fmt.Sprintf(
			`deleted_at IS NULL AND
			 u.user_id IN (
			     SELECT ue2.user_id FROM user_emails ue2
				 WHERE LOWER(ue2.email) IN (%s))`,
			utils.GenerateArray(0, len(emails)),
		),
		"limit": 1,
	}

	if err := s.selectTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	params := []any{}

	for _, email := range emails {
		params = append(params, email)
	}

	users, err := s.selectUsers(context.TODO(), wr.String(), params...)

	if err != nil || len(users) == 0 {
		return nil, err
	}

	return users[0], nil
}

// Admins will lists admin users.
func (s *Store) Admins(ctx context.Context) ([]*User, error) {
	var wr bytes.Buffer

	data := map[string]any{
		"where": "deleted_at IS NULL AND is_admin = TRUE",
		"limit": 50,
	}

	if err := s.selectTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	return s.selectUsers(ctx, wr.String())
}

func (s *Store) selectLicense(ctx context.Context, query string, params ...any) (*admin.License, error) {
	license := admin.License{}

	row, err := s.QueryRow(ctx, query, params...)

	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, nil
	}

	err = row.Scan(&license.Key, &license.Version, &license.Seats, &license.UserID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	license.Key = utils.DecryptToString(license.Key)
	return &license, nil
}

// LicenseByToken returns the license key by the given token.
func (s *Store) LicenseByToken(ctx context.Context, token string) (*admin.License, error) {
	pieces := strings.SplitN(token, ":", 2)

	if len(pieces) != 2 {
		return nil, errors.New("invalid-token")
	}

	userID, key := utils.StringToID(pieces[0]), pieces[1]
	license, err := s.LicenseByUserID(ctx, userID)

	if err != nil || license == nil {
		return nil, err
	}

	if license.Key != key {
		return nil, errors.New("invalid-token")
	}

	return license, nil
}

// LicenseByUserID returns the license key by the given user id.
func (s *Store) LicenseByUserID(ctx context.Context, userID types.ID) (*admin.License, error) {
	var wr bytes.Buffer

	data := map[string]any{
		"where": "user_id = $1",
	}

	if err := s.selectLicenseTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	return s.selectLicense(ctx, wr.String(), userID)
}

// InsertEmails inserts given email addresses for the user.
func (s *Store) InsertEmails(ctx context.Context, userID types.ID, emails []oauth.Email) error {
	var qb strings.Builder

	records := []map[string]int{}
	data := map[string]any{}

	params := []any{}
	counter := len(params)

	for _, email := range emails {
		records = append(records, utils.GenerateRecordRow(3, &counter))
		params = append(params, userID, email.Address, email.IsVerified, email.IsPrimary)
	}

	data["records"] = records

	if err := s.insertEmailsTmpl.Execute(&qb, data); err != nil {
		slog.Errorf("error executing update emails query template: %v", err)
		return err
	}

	_, err := s.Exec(ctx, qb.String(), params...)
	return err
}

// InsertUser inserts a user into the database.
func (s *Store) insertUser(user *User) (*User, error) {
	var wr bytes.Buffer

	emails := []map[string]int{}
	data := map[string]any{}

	params := []any{
		user.FirstName, user.LastName, user.DisplayName,
		user.Avatar, user.Metadata, user.IsAdmin, user.IsApproved,
		team.DEFAULT_TEAM_NAME, slug.Make(team.DEFAULT_TEAM_NAME),
	}

	counter := len(params)

	for _, email := range user.Emails {
		emails = append(emails, utils.GenerateRecordRow(3, &counter))
		params = append(params, email.Address, email.IsVerified, email.IsPrimary)
	}

	data["records"] = emails

	if err := s.insertTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	row, err := s.QueryRow(context.TODO(), wr.String(), params...)

	if err != nil {
		return nil, err
	}

	err = row.Scan(&user.ID)

	if err == nil {
		go sync3rdParties(user)
	}

	return user, err
}

// Accounts returns the list of connected accounts.
func (s *Store) Accounts(userID types.ID) ([]*ConnectedAccount, error) {
	accounts := []*ConnectedAccount{}

	rows, err := s.Query(context.TODO(), ustmt.selectAccounts, userID)

	if err != nil {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	for rows.Next() {
		var pac null.String
		acc := &ConnectedAccount{}

		if err := rows.Scan(&acc.Provider, &acc.URL, &acc.DisplayName, &pac); err != nil {
			slog.Error(err.Error())
			continue
		}

		acc.HasPersonalAccessToken = !pac.IsZero()
		accounts = append(accounts, acc)
	}

	return accounts, rows.Err()
}

type UserMetrics struct {
	BuildMinutes         int64
	FunctionInvocations  int64
	BandwidthUsedInBytes int64
	StorageUsedInBytes   int64
	UserID               types.ID
	Metadata             UserMeta
}

// HasBuildMinutes checks if the user has any remaining build minute left.
func (um UserMetrics) HasBuildMinutes() bool {
	limit, ok := config.Limits[um.Metadata.PackageName]
	seats := int64(utils.GetInt(um.Metadata.SeatsPurchased, 1))

	if !ok {
		return config.Limits[config.PackageFree].BuildMinutes*seats > um.BuildMinutes
	}

	return limit.BuildMinutes*seats > um.BuildMinutes
}

type UserMetricsArgs struct {
	UserID types.ID
	AppID  types.ID
}

// UserMetrics returns usage metrics for the given user.
func (s *Store) UserMetrics(ctx context.Context, args UserMetricsArgs) (*UserMetrics, error) {
	var wr bytes.Buffer

	data := map[string]any{}
	params := []any{}

	if args.UserID != 0 {
		data["where"] = "$1"
		params = append(params, args.UserID)
	} else if args.AppID != 0 {
		data["where"] = "(SELECT t.user_id FROM apps a JOIN teams t ON a.team_id = t.team_id WHERE a.app_id = $1)"
		params = append(params, args.AppID)
	} else {
		return &UserMetrics{}, nil
	}

	if err := s.userMetricsTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	row, err := s.QueryRow(ctx, wr.String(), params...)

	if err != nil {
		return nil, err
	}

	metrics := &UserMetrics{}

	err = row.Scan(
		&metrics.BandwidthUsedInBytes,
		&metrics.FunctionInvocations,
		&metrics.StorageUsedInBytes,
		&metrics.BuildMinutes,
		&metrics.UserID,
		&metrics.Metadata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	if metrics.Metadata.PackageName == "" {
		metrics.Metadata.PackageName = config.PackageFree
	}

	return metrics, nil
}

// MustUser inserts an auth user if it is not inserted yet.
func (s *Store) MustUser(authUser *oauth.User) (*User, error) {
	var user *User
	var err error
	ctx := context.TODO()

	emails := []string{}

	for _, email := range authUser.Emails {
		emails = append(emails, email.Address)
	}

	// Let's look for the user in the db
	user, err = s.UserByEmail(ctx, emails)

	if err != nil {
		return nil, err
	}

	// Let's make sure that emails are the same
	if user != nil {
		if !emailsEqual(user.Emails, authUser.Emails) {
			if err := s.InsertEmails(ctx, user.ID, authUser.Emails); err != nil {
				return nil, err
			}
		}
	}

	isSelfHosted := config.IsSelfHosted()

	// Otherwise let's create one
	if user == nil || user.ID == 0 {
		if isSelfHosted {
			count, err := s.SelectTotalUsers(ctx)

			if err != nil {
				return nil, err
			}

			// Check whether there is still enough room for more seats
			license := admin.CurrentLicense()

			if license.IsEnterprise() && int64(license.Seats) <= count {
				return nil, errors.New("seats-full")
			}
		}

		cnf := admin.MustConfig()
		signUpStatus := null.BoolFrom(true)

		switch cnf.SignUpMode() {
		case admin.SIGNUP_MODE_OFF:
			return nil, errors.New("pending-approval-or-rejected")
		case admin.SIGNUP_MODE_WAITLIST:
			signUpStatus = null.Bool{}

			for _, email := range authUser.Emails {
				if cnf.IsUserWhitelisted(email.Address) {
					signUpStatus = null.BoolFrom(true)
					break
				}
			}
		}

		user = &User{
			Emails:      authUser.Emails,
			Avatar:      null.NewString(authUser.AvatarURI, authUser.AvatarURI != ""),
			DisplayName: authUser.DisplayName,
			IsAdmin:     authUser.IsAdmin,
			IsApproved:  signUpStatus,
			IsNew:       true,
		}

		if strings.Contains(authUser.FullName, " ") {
			pieces := strings.Split(authUser.FullName, " ")
			user.FirstName = null.NewString(pieces[0], true)
			user.LastName = null.NewString(strings.Join(pieces[1:], " "), true)
		}

		if _, err = s.insertUser(user); err != nil {
			return nil, err
		}

		if _, err := s.createAPIKey(ctx, user.ID); err != nil {
			return nil, err
		}
	}

	return user, err
}

func (s *Store) UpdateLastLogin(ctx context.Context, userID types.ID) error {
	_, err := s.Exec(ctx, ustmt.updateLastLogin, userID)
	return err
}

// createAPIKey creates an API key for the user which is going to be used
// for interacting with the API for generic routes.
func (s *Store) createAPIKey(ctx context.Context, userID types.ID) (*apikey.Token, error) {
	token := &apikey.Token{
		UserID: userID,
		Name:   "default",
		Scope:  apikey.SCOPE_USER,
		Value:  apikey.GenerateTokenValue(),
	}

	if err := apikey.NewStore().AddAPIKey(ctx, token); err != nil {
		return nil, err
	}

	return token, nil
}

// SelectTotalUsers returns the total number of users in the instance.
func (s *Store) SelectTotalUsers(ctx context.Context) (int64, error) {
	var count int64

	row, err := s.QueryRow(ctx, ustmt.selectTotalUsers)

	if err != nil {
		return 0, err
	}

	if err = row.Scan(&count); err == sql.ErrNoRows {
		return 0, nil
	}

	return count, err
}

// SelectTotalUsersCloud returns the total number of users that are part of the
// teams the user owns.
func (s *Store) SelectTotalUsersCloud(ctx context.Context, userID types.ID) (int64, error) {
	var count int64

	row, err := s.QueryRow(ctx, ustmt.selectTotalUsersCloud, userID)

	if err != nil {
		return 0, err
	}

	if err = row.Scan(&count); err == sql.ErrNoRows {
		return 0, nil
	}

	return count, err
}

// MarkUserAsDeleted marks the user, apps and deployment as deleted.
func (s *Store) MarkUserAsDeleted(context context.Context, userID types.ID) error {
	tx, err := s.Conn.Begin()

	if err != nil {
		return err
	}

	query, err := tx.Prepare(ustmt.markUserAsDeleted)

	if err != nil {
		return err
	}

	if _, err := query.Exec(userID); err != nil {
		_ = tx.Rollback()
		return err
	}

	query, err = tx.Prepare(ustmt.markUserAppsAsDeleted)

	if err != nil {
		return err
	}

	var appIds []int64

	{
		rows, err := query.QueryContext(context, userID)

		if err != nil {
			_ = tx.Rollback()
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id int64
			err := rows.Scan(&id)
			if err != nil {
				_ = tx.Rollback()
				return err
			}
			appIds = append(appIds, id)
		}
	}

	query, err = tx.PrepareContext(context, ustmt.markDeploymentsAsDeleted)

	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = query.ExecContext(context, pq.Array(appIds))

	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func sync3rdParties(user *User) {
	go discord.Notify(config.Get().Reporting.DiscordSignupsChannel, discord.Payload{
		Embeds: []discord.PayloadEmbed{
			{
				Title:     "New Signup",
				Timestamp: time.Now().Format(time.RFC3339),
				Fields: []discord.PayloadField{
					{Name: "ID", Value: strconv.FormatInt(int64(user.ID), 10)},
					{Name: "Name", Value: user.FullName()},
					{Name: "Email", Value: user.PrimaryEmail()},
				}},
		},
	})
}

type SubscriptionArgs struct {
	UserID           types.ID
	CustomerID       string
	SubscriptionTier string
	Quantity         int64
}

// UpdateSubscription updates the package name for the user. If it fails,
// it will post a message.
func (s *Store) UpdateSubscription(ctx context.Context, userID types.ID, meta UserMeta) error {
	_, err := s.Exec(ctx, ustmt.updateSubscription, meta, userID)

	if err != nil {
		return err
	}

	// Remove license if the package is not premium or ultimate
	if meta.PackageName != config.PackagePremium && meta.PackageName != config.PackageUltimate {
		_, err = s.Exec(ctx, ustmt.deleteLicense, userID)
		return err
	}

	return err
}

// GenerateSelfHostedLicense will generate a license with the user's api key. If the user
// has an api key it will be used, otherwise a new api key will be generated.
func (s *Store) GenerateSelfHostedLicense(ctx context.Context, quantity int, userID types.ID, meta map[string]any) (*admin.License, error) {
	// Generate a new license using the apiKey.value
	license := admin.NewLicense(admin.NewLicenseArgs{
		Seats:    quantity,
		UserID:   userID,
		Metadata: meta,
	})

	md, err := json.Marshal(license.Metadata)

	if err != nil {
		return nil, err
	}

	// Make sure to delete the previous license
	if _, err = s.Exec(ctx, ustmt.deleteLicense, userID); err != nil {
		return nil, err
	}

	_, err = s.Exec(ctx, ustmt.insertLicense,
		utils.EncryptToString(license.Key),
		license.Version,
		license.Seats,
		null.NewInt(int64(userID), userID > 0),
		md,
	)

	if err != nil {
		return nil, err
	}

	return license, nil
}

// UpdatePersonalAccessToken updates the personal access token for the
// given user id.
func (s *Store) UpdatePersonalAccessToken(uid types.ID, token string) (err error) {
	if token == "" {
		_, err = s.Exec(context.TODO(), ustmt.updatePersonalAccessToken, nil, uid)
		return err
	}

	encrypted, err := utils.Encrypt([]byte(token))

	if err != nil {
		return err
	}

	_, err = s.Exec(context.TODO(), ustmt.updatePersonalAccessToken, encrypted, uid)
	return err
}

// emailsEqual checks the equality between two oauth.Email slices.
// This function does not consider the `IsPrimary` field because
// we allow users modifying their primary emails.
func emailsEqual(emailsA, emailsB []oauth.Email) bool {
	if len(emailsA) != len(emailsB) {
		return false
	}

	a := map[string]bool{}

	for _, email := range emailsA {
		a[strings.ToLower(email.Address)] = email.IsVerified
	}

	for _, email := range emailsB {
		isVerified, ok := a[strings.ToLower(email.Address)]

		if !ok {
			return false
		}

		if isVerified != email.IsVerified {
			return false
		}
	}

	return true
}

type Usage struct {
	UserID              types.ID
	FunctionInvocations int64
	BandwidthInBytes    int64
}

// UpdateUsageMetrics updates the bandwidth and function invocations for the given user.
func (s *Store) UpdateUsageMetrics(ctx context.Context, usage []Usage) error {
	if len(usage) == 0 {
		return nil
	}

	var qb strings.Builder

	if err := s.updateUserMetricsTmpl.Execute(&qb, usage); err != nil {
		slog.Errorf("error executing batch query template: %v", err)
		return err
	}

	params := []any{}
	now := time.Now().UTC()
	month := now.Month()
	year := now.Year()

	for _, record := range usage {
		params = append(params,
			record.UserID, record.BandwidthInBytes,
			record.FunctionInvocations, year, month,
		)
	}

	_, err := s.Exec(ctx, qb.String(), params...)
	return err
}
