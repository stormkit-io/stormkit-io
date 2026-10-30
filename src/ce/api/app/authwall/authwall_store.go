package authwall

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/crypto/bcrypt"
)

// envApp selects the app of the environment $1 and the app's team. Apps
// created before teams may have no team_id; only those fall back to the
// owner's personal team.
const envApp = `
	SELECT
		a.app_id, COALESCE(a.team_id, t.team_id) AS team_id
	FROM
		apps_build_conf e
	JOIN
		apps a ON a.app_id = e.app_id
	LEFT JOIN
		teams t ON t.user_id = a.user_id AND t.is_default IS TRUE
	WHERE
		e.env_id = $1
`

// opensApp is true when the login aw opens the wall of the app ea.
const opensApp = `
	aw.team_id = ea.team_id AND (
		aw.all_apps OR EXISTS (
			SELECT 1 FROM auth_wall_apps awa
			WHERE awa.login_id = aw.login_id AND awa.app_id = ea.app_id
		)
	)
`

var stmt = struct {
	selectTeamLogins  string
	updateLogin       string
	deleteLogins      string
	countTeamApps     string
	selectTeamEnvs    string
	createLogin       string
	selectLogin       string
	updatePassword    string
	updateLastLogin   string
	setAuthWallConfig string
}{
	selectTeamLogins: `
		SELECT
			aw.login_id, aw.team_id, aw.login_email, aw.login_metadata,
			aw.all_apps, aw.last_login_at, aw.created_at,
			COALESCE(ARRAY_AGG(ap.app_id ORDER BY ap.app_id) FILTER (WHERE ap.app_id IS NOT NULL), '{}')
		FROM
			auth_wall aw
		LEFT JOIN
			auth_wall_apps awa ON awa.login_id = aw.login_id
		LEFT JOIN
			apps ap ON ap.app_id = awa.app_id AND ap.deleted_at IS NULL
		WHERE
			aw.team_id = $1 AND
			($2 = 0 OR aw.login_id = $2)
		GROUP BY
			aw.login_id
		ORDER BY
			LOWER(aw.login_email) ASC
		LIMIT $3 OFFSET $4;
	`,

	// The access rows are replaced only when $5 is true. Statements in a WITH
	// see the rows as they were before it, so the delete and insert do not see
	// each other.
	updateLogin: `
		WITH login AS (
			UPDATE auth_wall SET
				login_password = COALESCE(NULLIF($3, ''), login_password),
				login_metadata = $4,
				all_apps = $6
			WHERE
				login_id = $1 AND team_id = $2
			RETURNING login_id
		), removed AS (
			DELETE FROM auth_wall_apps
			WHERE
				$5 AND
				login_id IN (SELECT login_id FROM login) AND
				app_id <> ALL($7::bigint[])
		), added AS (
			INSERT INTO auth_wall_apps (login_id, app_id)
			SELECT login.login_id, UNNEST($7::bigint[]) FROM login WHERE $5
			ON CONFLICT DO NOTHING
		)
		SELECT COUNT(*) FROM login;
	`,

	deleteLogins: `
		DELETE FROM auth_wall WHERE team_id = $1 AND login_id = ANY($2)
		RETURNING login_id;
	`,

	countTeamApps: `
		SELECT COUNT(*) FROM apps
		WHERE team_id = $1 AND deleted_at IS NULL AND app_id = ANY($2);
	`,

	selectTeamEnvs: `
		SELECT
			a.app_id, COALESCE(a.display_name, ''), e.env_id, e.env_name,
			COALESCE(e.auth_wall_conf->>'status', ''),
			COALESCE(e.auth_wall_conf->'publicPaths', '[]'::jsonb)
		FROM
			apps a
		JOIN
			apps_build_conf e ON e.app_id = a.app_id
		WHERE
			a.team_id = $1 AND
			($2 = 0 OR e.env_id = $2) AND
			a.deleted_at IS NULL AND
			e.deleted_at IS NULL
		ORDER BY
			LOWER(a.display_name) ASC, a.app_id ASC, e.env_name ASC;
	`,

	createLogin: `
		WITH login AS (
			INSERT INTO auth_wall
				(team_id, login_email, login_password, login_metadata, all_apps, created_at)
			VALUES
				($1, $2, $3, $4, $5, NOW())
			RETURNING
				login_id
		), access AS (
			INSERT INTO auth_wall_apps (login_id, app_id)
			SELECT login.login_id, UNNEST($6::bigint[]) FROM login
		)
		SELECT login_id FROM login;
	`,

	selectLogin: `
		WITH ea AS (` + envApp + `)
		SELECT
			aw.login_id, aw.login_email, aw.login_password, aw.login_metadata
		FROM
			auth_wall aw, ea
		WHERE
			LOWER(aw.login_email) = LOWER($2) AND ` + opensApp + `;
	`,

	updatePassword: `
		UPDATE auth_wall SET login_password = $1 WHERE login_id = $2;
	`,

	updateLastLogin: `
		UPDATE auth_wall SET last_login_at = NOW() AT TIME ZONE 'UTC' WHERE login_id = $1;
	`,

	setAuthWallConfig: `
		UPDATE apps_build_conf SET auth_wall_conf = $1 WHERE env_id = $2;
	`,
}

// dummyHash is compared against when no login matches the email, so a failed
// login takes as long whether or not the email exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("auth-wall-dummy-password"), bcrypt.DefaultCost)

type store struct {
	*database.Store
}

// Store returns a store instance.
func Store() *store {
	return &store{database.NewStore()}
}

// CreateLogin creates a login in aw.TeamID that opens the apps in aw.AppIDs,
// or every app of the team when aw.AllApps is set. The password is stored as a
// bcrypt hash.
func (s *store) CreateLogin(ctx context.Context, aw *AuthWall) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(aw.LoginPassword), bcrypt.DefaultCost)

	if err != nil {
		return err
	}

	appIDs := aw.AppIDs

	if appIDs == nil {
		appIDs = []types.ID{}
	}

	params := []any{
		aw.TeamID,
		aw.LoginEmail,
		string(hash),
		aw.Metadata,
		aw.AllApps,
		pq.Array(appIDs),
	}

	row, err := s.QueryRow(ctx, stmt.createLogin, params...)

	if err != nil {
		return err
	}

	if row == nil {
		return errors.New("no row returned")
	}

	return row.Scan(&aw.LoginID)
}

// LoginParams are the credentials submitted to the wall of an environment.
type LoginParams struct {
	EnvID    types.ID
	Email    string
	Password string
}

// Login returns the login matching the credentials when it opens the wall of
// the environment, and nil otherwise.
func (s *store) Login(ctx context.Context, p LoginParams) (*AuthWall, error) {
	row, err := s.QueryRow(ctx, stmt.selectLogin, p.EnvID, p.Email)

	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, nil
	}

	aw := &AuthWall{}
	err = row.Scan(&aw.LoginID, &aw.LoginEmail, &aw.LoginPassword, &aw.Metadata)

	if errors.Is(err, sql.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(p.Password))
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if !s.verifyPassword(ctx, aw, p.Password) {
		return nil, nil
	}

	return aw, nil
}

// verifyPassword compares the password with the stored one. Logins created
// before passwords were hashed store them encrypted; those are hashed on the
// first successful login.
func (s *store) verifyPassword(ctx context.Context, aw *AuthWall, password string) bool {
	if strings.HasPrefix(aw.LoginPassword, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(aw.LoginPassword), []byte(password)) == nil
	}

	if subtle.ConstantTimeCompare([]byte(utils.DecryptToString(aw.LoginPassword)), []byte(password)) != 1 {
		return false
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err == nil {
		_, err = s.Exec(ctx, stmt.updatePassword, string(hash), aw.LoginID)
	}

	if err != nil {
		slog.Errorf("error while hashing auth wall password: %s", err.Error())
	}

	return true
}

// TeamLoginsParams are the parameters of TeamLogins.
type TeamLoginsParams struct {
	TeamID types.ID

	// LoginID limits the result to one login.
	LoginID types.ID
	From    int
	Limit   int
}

// TeamLogins returns the logins of a team, ordered by email, with the apps
// they open. Deleted apps are left out.
func (s *store) TeamLogins(ctx context.Context, p TeamLoginsParams) ([]AuthWall, error) {
	rows, err := s.Query(ctx, stmt.selectTeamLogins, p.TeamID, p.LoginID, p.Limit, p.From)

	if err != nil || rows == nil {
		return nil, err
	}

	defer rows.Close()

	logins := []AuthWall{}

	for rows.Next() {
		aw := AuthWall{}
		appIDs := pq.Int64Array{}

		err := rows.Scan(
			&aw.LoginID, &aw.TeamID, &aw.LoginEmail, &aw.Metadata,
			&aw.AllApps, &aw.LastLogin, &aw.CreatedAt, &appIDs,
		)

		if err != nil {
			return nil, err
		}

		aw.AppIDs = make([]types.ID, len(appIDs))

		for i, id := range appIDs {
			aw.AppIDs[i] = types.ID(id)
		}

		logins = append(logins, aw)
	}

	return logins, rows.Err()
}

// TeamLogin returns a login of the team, or nil when the team has no such
// login.
func (s *store) TeamLogin(ctx context.Context, teamID, loginID types.ID) (*AuthWall, error) {
	logins, err := s.TeamLogins(ctx, TeamLoginsParams{TeamID: teamID, LoginID: loginID, Limit: 1})

	if err != nil || len(logins) == 0 {
		return nil, err
	}

	return &logins[0], nil
}

// UpdateLoginParams are the parameters of UpdateLogin.
type UpdateLoginParams struct {
	// Login holds the login's new metadata and access. Its LoginPassword is
	// ignored.
	Login *AuthWall

	// Password is the new password. Empty keeps the current one.
	Password string

	// ReplaceApps replaces the apps the login opens with Login.AppIDs.
	ReplaceApps bool
}

// UpdateLogin saves the changes to a login of Login.TeamID. It returns false
// when the team has no such login.
func (s *store) UpdateLogin(ctx context.Context, p UpdateLoginParams) (bool, error) {
	hash := ""

	if p.Password != "" {
		b, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)

		if err != nil {
			return false, err
		}

		hash = string(b)
	}

	appIDs := p.Login.AppIDs

	if appIDs == nil {
		appIDs = []types.ID{}
	}

	row, err := s.QueryRow(
		ctx, stmt.updateLogin,
		p.Login.LoginID, p.Login.TeamID, hash, p.Login.Metadata,
		p.ReplaceApps, p.Login.AllApps, pq.Array(appIDs),
	)

	if err != nil || row == nil {
		return false, err
	}

	var count int

	if err := row.Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

// DeleteLoginsParams are the parameters of DeleteLogins.
type DeleteLoginsParams struct {
	TeamID   types.ID
	LoginIDs []types.ID
}

// DeleteLogins deletes logins of a team and returns the IDs it deleted. IDs of
// other teams' logins are ignored.
func (s *store) DeleteLogins(ctx context.Context, p DeleteLoginsParams) ([]types.ID, error) {
	rows, err := s.Query(ctx, stmt.deleteLogins, p.TeamID, pq.Array(p.LoginIDs))

	if err != nil || rows == nil {
		return nil, err
	}

	defer rows.Close()

	deleted := []types.ID{}

	for rows.Next() {
		var id types.ID

		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		deleted = append(deleted, id)
	}

	return deleted, rows.Err()
}

// AreTeamApps reports whether every app in appIDs belongs to the team. The IDs
// must be unique.
func (s *store) AreTeamApps(ctx context.Context, teamID types.ID, appIDs []types.ID) (bool, error) {
	if len(appIDs) == 0 {
		return true, nil
	}

	row, err := s.QueryRow(ctx, stmt.countTeamApps, teamID, pq.Array(appIDs))

	if err != nil || row == nil {
		return false, err
	}

	var count int

	if err := row.Scan(&count); err != nil {
		return false, err
	}

	return count == len(appIDs), nil
}

// TeamEnvsParams are the parameters of TeamEnvs.
type TeamEnvsParams struct {
	TeamID types.ID

	// EnvID limits the result to one environment.
	EnvID types.ID
}

// TeamEnvs returns the environments of a team's apps with the mode of their
// walls.
func (s *store) TeamEnvs(ctx context.Context, p TeamEnvsParams) ([]TeamEnv, error) {
	rows, err := s.Query(ctx, stmt.selectTeamEnvs, p.TeamID, p.EnvID)

	if err != nil || rows == nil {
		return nil, err
	}

	defer rows.Close()

	envs := []TeamEnv{}

	for rows.Next() {
		env := TeamEnv{}
		publicPaths := []byte{}

		if err := rows.Scan(&env.AppID, &env.AppName, &env.EnvID, &env.EnvName, &env.Status, &publicPaths); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(publicPaths, &env.PublicPaths); err != nil {
			return nil, err
		}

		envs = append(envs, env)
	}

	return envs, rows.Err()
}

// UpdateLastLogin updates the last login time.
func (s *store) UpdateLastLogin(ctx context.Context, loginID types.ID) error {
	_, err := s.Exec(ctx, stmt.updateLastLogin, loginID)
	return err
}

func (s *store) SetAuthWallConfig(ctx context.Context, envID types.ID, cfg *Config) error {
	_, err := s.Exec(ctx, stmt.setAuthWallConfig, cfg, envID)
	return err
}

// AuthWallConfig returns the configuration associated with the environment.
func (s *store) AuthWallConfig(ctx context.Context, envID types.ID) (*Config, error) {
	row, err := s.QueryRow(ctx, "SELECT auth_wall_conf FROM apps_build_conf WHERE env_id = $1", envID)

	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, nil
	}

	cfg := &Config{}

	if err := row.Scan(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
