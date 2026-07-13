package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"gopkg.in/guregu/null.v3"
)

type queries struct {
	insertAudit  string
	selectAudits string
}

var stmts = queries{
	insertAudit: `
		INSERT INTO audit_logs (
			audit_action, audit_diff, 
			team_id, app_id, env_id, user_id, 
			user_display, token_name
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING
			audit_id;
	`,

	selectAudits: `
		SELECT
			a.audit_id,
			a.audit_action,
			a.audit_diff,
			a.team_id,
			a.app_id,
			a.env_id,
			a.user_id,
			a.user_display,
			a.token_name,
			a.created_at,
			env.env_name
		FROM
			audit_logs a
		LEFT JOIN
			apps_build_conf env ON env.env_id = a.env_id
		WHERE
			{{ .where }}
		ORDER BY
			a.audit_id DESC
		LIMIT
			{{ or .limit 100 }};
	`,
}

type Store struct {
	*database.Store

	selectTmpl *template.Template
}

func NewStore() *Store {
	selectTmpl, err := template.New("selectAudits").Parse(stmts.selectAudits)

	if err != nil {
		panic(err)
	}

	return &Store{
		Store:      database.NewStore(),
		selectTmpl: selectTmpl,
	}
}

func nilOrValue(id types.ID) *types.ID {
	if id > 0 {
		return &id
	}

	return nil
}

// Log logs an audit entry into the audit log table.
func (s *Store) Log(ctx context.Context, audit *Audit) error {
	var diff any
	var err error

	if audit.Diff != nil {
		diff, err = json.Marshal(audit.Diff)

		if err != nil {
			return err
		}
	} else {
		diff = nil
	}

	row, err := s.QueryRow(
		context.TODO(),
		stmts.insertAudit,
		audit.Action, diff,
		nilOrValue(types.ID(audit.TeamID)),
		nilOrValue(audit.AppID),
		nilOrValue(audit.EnvID),
		nilOrValue(audit.UserID),
		audit.UserDisplay,
		audit.TokenName,
	)

	if err != nil {
		return err
	}

	return row.Scan(&audit.ID)
}

type AuditFilters struct {
	AppID    types.ID
	EnvID    types.ID
	TeamID   types.ID
	BeforeID types.ID
	Limit    int
}

func (s *Store) SelectAudits(ctx context.Context, filters AuditFilters) ([]Audit, error) {
	var wr bytes.Buffer

	where := []string{}
	params := []any{}

	if filters.AppID > 0 {
		params = append(params, filters.AppID)
		where = append(where, fmt.Sprintf("a.app_id = $%d", len(params)))
	}

	if filters.EnvID > 0 {
		params = append(params, filters.EnvID)
		where = append(where, fmt.Sprintf("a.env_id = $%d", len(params)))
	}

	if filters.TeamID > 0 {
		params = append(params, filters.TeamID)
		where = append(where, fmt.Sprintf("a.team_id = $%d", len(params)))
	}

	if filters.BeforeID > 0 {
		params = append(params, filters.BeforeID)
		where = append(where, fmt.Sprintf("a.audit_id < $%d", len(params)))
	}

	if filters.Limit == 0 {
		filters.Limit = 100
	}

	data := map[string]any{
		"where": strings.Join(where, " AND "),
		"limit": filters.Limit + 1,
	}

	if err := s.selectTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	audits := []Audit{}

	rows, err := s.Query(ctx, wr.String(), params...)

	if err != nil || rows == nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		audit := Audit{}
		diff := []byte{}

		var teamID null.Int
		var appID null.Int
		var envID null.Int
		var userID null.Int
		var envName null.String

		err := rows.Scan(
			&audit.ID, &audit.Action,
			&diff, &teamID, &appID,
			&envID, &userID, &audit.UserDisplay,
			&audit.TokenName, &audit.Timestamp,
			&envName,
		)

		audit.UserID = types.ID(userID.ValueOrZero())
		audit.AppID = types.ID(appID.ValueOrZero())
		audit.EnvID = types.ID(envID.ValueOrZero())
		audit.TeamID = types.ID(teamID.ValueOrZero())
		audit.EnvName = envName.ValueOrZero()

		if err != nil {
			return nil, err
		}

		if diff != nil {
			audit.Diff = &Diff{}

			if err := json.Unmarshal(diff, audit.Diff); err != nil {
				return nil, err
			}
		}

		audits = append(audits, audit)
	}

	return audits, nil
}
