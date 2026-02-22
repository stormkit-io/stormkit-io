package team

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

var tableTeams = "teams"
var tableApps = "apps"
var tableTeamMembers = "team_members"

const (
	ROLE_OWNER         = "owner"
	ROLE_ADMIN         = "admin"
	ROLE_DEVELOPER     = "developer"
	DEFAULT_TEAM_NAME  = "Personal"
	MAX_TEAMS_PER_USER = 25
)

var sqlTemplates = struct {
	selectTeamMembers *template.Template
}{
	selectTeamMembers: template.Must(template.New("selectTeamMembers").Parse(`
		SELECT
			u.user_id, u.first_name, u.last_name, u.display_name, 
			(SELECT ue.email FROM user_emails ue WHERE ue.user_id = u.user_id AND ue.is_primary IS TRUE LIMIT 1),
			tm.member_id, tm.member_role, tm.membership_status
		FROM team_members tm
		LEFT JOIN users u ON u.user_id = tm.user_id
		WHERE
			{{ .where }}
		LIMIT 100;
	`)),
}

var stmt = struct {
	createTeam            string
	updateTeam            string
	addUserToTeam         string
	selectTeam            string
	selectTeams           string
	selectDefaultTeam     string
	isTeamMember          string
	markTeamAsSoftDeleted string
	removeUserFromTeam    string
	migrateApp            string
}{
	createTeam: `
		WITH new_team AS (
			INSERT INTO teams (team_name, team_slug, user_id, is_default, created_at)
			VALUES ($1, $2, $3, FALSE, NOW())
			RETURNING team_id
		)
		INSERT INTO team_members (team_id, user_id, member_role, membership_status)
		SELECT team_id, $3, $4, $5 FROM new_team
		RETURNING team_id, member_id;
	`,

	updateTeam: fmt.Sprintf(`
		UPDATE %s SET team_name = $1, team_slug = $2 WHERE team_id = $3;
	`, tableTeams),

	selectTeam: `
		SELECT
			t.team_id, t.is_default,
			t.team_name, t.team_slug, tm.member_role
		FROM teams t
		LEFT JOIN team_members tm ON tm.team_id = t.team_id
		WHERE
			t.team_id = $1 AND
			t.deleted_at IS NULL AND
			tm.user_id = $2 AND
			tm.membership_status IS TRUE;
	`,

	selectTeams: fmt.Sprintf(`
		SELECT 
			t.team_id, t.is_default,
			t.team_name, t.team_slug, tm.member_role
		FROM %s t
		LEFT JOIN %s tm ON tm.team_id = t.team_id
		WHERE
			t.deleted_at IS NULL AND
			tm.user_id = $1 AND
			tm.membership_status IS TRUE
		ORDER BY t.team_id ASC
		LIMIT 25;
	`, tableTeams, tableTeamMembers),

	selectDefaultTeam: fmt.Sprintf(`
		SELECT
			t.team_id, t.is_default,
			t.team_name, t.team_slug, tm.member_role
		FROM %s t
		LEFT JOIN %s tm ON tm.team_id = t.team_id
		WHERE 
			tm.user_id = $1 AND 
			tm.membership_status IS TRUE AND
			tm.member_role = 'owner' AND
			t.is_default IS TRUE
	`, tableTeams, tableTeamMembers),

	addUserToTeam: fmt.Sprintf(`
		INSERT INTO %s (team_id, user_id, member_role, membership_status)
		VALUES ($1, $2, $3, $4)
		RETURNING member_id;
	`, tableTeamMembers),

	isTeamMember: fmt.Sprintf(`
		SELECT COUNT(*) FROM %s tm
		WHERE
			tm.user_id = $1 AND
			tm.team_id = $2 AND
			membership_status IS TRUE
		LIMIT 1;
	`, tableTeamMembers),

	markTeamAsSoftDeleted: `
		UPDATE teams SET deleted_at = NOW() WHERE team_id = $1;
	`,

	removeUserFromTeam: fmt.Sprintf(`
		DELETE FROM %s
		WHERE
			team_id = $1 AND
			member_id = $2 AND
			member_role <> $3;
	`, tableTeamMembers),

	migrateApp: fmt.Sprintf(`
		UPDATE %s
		SET team_id = $1
		WHERE app_id = $2;
	`, tableApps),
}

// Store handles user logic in the database.
type Store struct {
	*database.Store
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{database.NewStore()}
}

// CreateTeam creates a team and adds the user as an owner.
func (s *Store) CreateTeam(ctx context.Context, team *Team, member *Member) error {
	if member.Role != ROLE_OWNER {
		return errors.New("member needs to be an owner to create a team")
	}

	row, err := s.QueryRow(ctx, stmt.createTeam, team.Name, team.Slug, member.UserID, member.Role, member.Status)

	if err != nil {
		return err
	}

	if err = row.Scan(&team.ID, &member.ID); err != nil {
		return err
	}

	member.TeamID = team.ID
	return nil
}

// CreateTeam creates a team and adds the user as an owner.
func (s *Store) UpdateTeam(ctx context.Context, team *Team) error {
	_, err := s.Exec(ctx, stmt.updateTeam, team.Name, team.Slug, team.ID)
	return err
}

// Teams returns the teams that the user is a member of.
func (s *Store) Teams(ctx context.Context, userID types.ID) ([]Team, error) {
	teams := []Team{}

	rows, err := s.Query(ctx, stmt.selectTeams, userID)

	if err == sql.ErrNoRows {
		return teams, nil
	}

	if err != nil {
		slog.Errorf("error while fetching teams: %s", err.Error())
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		t := Team{}

		if err := rows.Scan(&t.ID, &t.IsDefault, &t.Name, &t.Slug, &t.CurrentUserRole); err != nil {
			slog.Errorf("error while scanning team row: %s", err.Error())
			return teams, err
		}

		teams = append(teams, t)
	}

	return teams, nil
}

// AddMemberToTeam adds the given member to the team.
func (s *Store) AddMemberToTeam(ctx context.Context, member *Member) error {
	row, err := s.QueryRow(ctx, stmt.addUserToTeam, member.TeamID, member.UserID, member.Role, member.Status)

	if err != nil {
		return err
	}

	return row.Scan(&member.ID)
}

// Team returns the team with the given id.
func (s *Store) Team(ctx context.Context, teamID, userID types.ID) (*Team, error) {
	t := &Team{}

	row, err := s.QueryRow(ctx, stmt.selectTeam, teamID, userID)

	if err != nil {
		return nil, err
	}

	err = row.Scan(&t.ID, &t.IsDefault, &t.Name, &t.Slug, &t.CurrentUserRole)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return t, err
}

// DefaultTeam returns the default team of the user.
func (s *Store) DefaultTeam(ctx context.Context, userID types.ID) (*Team, error) {
	t := &Team{}

	row, err := s.QueryRow(ctx, stmt.selectDefaultTeam, userID)

	if err != nil {
		return nil, err
	}

	err = row.Scan(&t.ID, &t.IsDefault, &t.Name, &t.Slug, &t.CurrentUserRole)

	if err != nil {
		return nil, err
	}

	return t, nil
}

// DefaultTeamID returns the default team id of the user.
func (s *Store) DefaultTeamID(ctx context.Context, userID types.ID) (types.ID, error) {
	t, err := s.DefaultTeam(ctx, userID)
	return t.ID, err
}

// MarkTeamAsSoftDeleted marks the given team as soft deleted.
func (s *Store) MarkTeamAsSofDeleted(ctx context.Context, teamID types.ID) error {
	_, err := s.Exec(ctx, stmt.markTeamAsSoftDeleted, teamID)
	return err
}

// IsMember checks if the user is a member of the given team.
func (s *Store) IsMember(ctx context.Context, userID types.ID, teamID types.ID) bool {
	var count int

	row, err := s.QueryRow(ctx, stmt.isTeamMember, userID, teamID)

	if err != nil {
		slog.Errorf("error while checking if user is a team member: %s", err.Error())
		return false
	}

	if err = row.Scan(&count); err != nil {
		slog.Errorf("error while scanning team member count: %s", err.Error())
		return false
	}

	return count > 0
}

type TeamMemberFilters struct {
	TeamID   types.ID
	MemberID types.ID
	Role     string
}

// TeamMember returns the member with the given id in the given team.
func (s *Store) TeamMember(ctx context.Context, teamID, memberID types.ID) (*Member, error) {
	filters := TeamMemberFilters{
		TeamID:   teamID,
		MemberID: memberID,
	}

	members, err := s.TeamMembers(ctx, filters)

	if err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return nil, nil
	}

	return &members[0], nil
}

// TeamMembers returns members for the given team.
func (s *Store) TeamMembers(ctx context.Context, filters TeamMemberFilters) ([]Member, error) {
	members := []Member{}
	where := []string{"tm.team_id = $1"}
	params := []any{filters.TeamID}
	buf := bytes.Buffer{}

	if filters.Role != "" {
		where = append(where, "tm.member_role = $2")
		params = append(params, filters.Role)
	}

	if filters.MemberID != 0 {
		where = append(where, fmt.Sprintf("tm.member_id = $%d", len(params)+1))
		params = append(params, filters.MemberID)
	}

	err := sqlTemplates.selectTeamMembers.Execute(&buf, map[string]string{
		"where": strings.Join(where, " AND "),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to render team members template: %w", err)
	}

	rows, err := s.Query(ctx, buf.String(), params...)

	if rows == nil && err == nil {
		return members, nil
	}

	if err != nil {
		slog.Errorf("[TeamMembers] query error: %s", err.Error())
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		m := Member{}
		err := rows.Scan(
			&m.UserID, &m.FirstName, &m.LastName,
			&m.DisplayName, &m.Email, &m.ID, &m.Role, &m.Status,
		)

		if err != nil {
			slog.Errorf("[TeamMembers] scan error: %s", err.Error())
			return members, err
		}

		members = append(members, m)
	}

	return members, nil
}

// RemoveTeamMember hard deletes the given member from the team.
func (s *Store) RemoveTeamMember(ctx context.Context, teamID, memberID types.ID) error {
	_, err := s.Exec(ctx, stmt.removeUserFromTeam, teamID, memberID, ROLE_OWNER)
	return err
}

// MigrateApp migrates the given app id to the given team.
func (s *Store) MigrateApp(ctx context.Context, appID, teamID types.ID) error {
	_, err := s.Exec(ctx, stmt.migrateApp, teamID, appID)
	return err
}
