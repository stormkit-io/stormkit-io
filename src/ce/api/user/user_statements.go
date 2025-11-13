package user

import (
	"fmt"
)

var (
	tableDeploys = "deployments"
)

type userStatement struct {
	selectUsers               string
	selectTotalUsers          string
	selectTotalUsersCloud     string
	selectAccounts            string
	insertEmails              string
	updateSubscription        string
	insertUser                string
	markUserAsDeleted         string
	markUserAppsAsDeleted     string
	markDeploymentsAsDeleted  string
	updatePersonalAccessToken string
	selectLicense             string
	insertLicense             string
	deleteLicense             string
	updateLastLogin           string
	userMetrics               string
	updateUsageMetrics        string
}

var ustmt = &userStatement{
	selectUsers: `
		SELECT
			u.user_id, u.avatar_uri, u.display_name, u.first_name,
			u.last_name,
			json_agg(
				jsonb_build_object(
					'address', ue.email,
					'primary', ue.is_primary,
					'verified', ue.is_verified
				)
				ORDER BY ue.email_id ASC
			) as emails,
			u.created_at, u.is_admin, u.metadata,
			u.last_login_at, u.is_approved
		FROM
			users u
		LEFT JOIN
			user_emails ue ON ue.user_id = u.user_id
		WHERE
			{{ .where }}
		GROUP BY
			u.user_id
		ORDER
			BY u.user_id DESC
		LIMIT
			{{ or .limit 50 }};
	`,

	markDeploymentsAsDeleted: fmt.Sprintf(`
		UPDATE %s
		SET
			deleted_at = NOW(),
			exit_code = COALESCE(exit_code, -1)
		WHERE
			app_id = ANY($1) AND
			deleted_at IS NULL;
	`, tableDeploys),

	selectTotalUsers: `
		SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;
	`,

	selectTotalUsersCloud: `
		WITH owned_teams AS (
    		SELECT DISTINCT
				team_id 
    		FROM
				skitapi.team_members 
    		WHERE
				user_id = $1 AND member_role = 'owner'
		)
		SELECT
			COUNT(distinct tm.user_id) as total_members
		FROM
			skitapi.team_members tm
		JOIN
			owned_teams ot ON tm.team_id = ot.team_id;
	`,

	selectAccounts: `
		SELECT
			provider, account_uri, display_name, personal_access_token
		FROM user_access_tokens
		WHERE
			user_id = $1
		ORDER BY provider;
	`,

	insertEmails: `
		WITH
			data(user_id, email, is_verified, is_primary) as (
				VALUES
					{{ generateValues 4 (len .records) }}
			)
			INSERT INTO user_emails (user_id, email, is_verified, is_primary)
				SELECT
					d.user_id::BIGINT, d.email, d.is_verified::BOOLEAN, d.is_primary::BOOLEAN
				FROM
					data d
				WHERE NOT EXISTS (
					SELECT 1 FROM user_emails ue
					WHERE LOWER(ue.email) = LOWER(d.email)
				)
	`,

	insertUser: `
		WITH
			new_user AS (
				INSERT INTO users
					(first_name, last_name, display_name, avatar_uri, metadata, is_admin, is_approved)
				VALUES
					($1, $2, $3, $4, $5, $6, $7)
				RETURNING user_id
			),
			new_team AS (
				INSERT INTO teams
					(team_name, team_slug, user_id, is_default, created_at)
				VALUES
					($8, $9, (SELECT user_id FROM new_user), TRUE, NOW())
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
			),
			data(user_id, email, is_verified, is_primary) as (
				VALUES {{ range $i, $record := .records }}
					(
						(SELECT user_id FROM new_user), ${{ $record.p1 }}, 
						${{ $record.p2 }}, ${{ $record.p3 }}
					){{ if not (last $i $.records) }}, {{ end }}
				{{ end }}
			)
			INSERT INTO user_emails (user_id, email, is_verified, is_primary)
				SELECT
					d.user_id::BIGINT, d.email, d.is_verified::BOOLEAN, d.is_primary::BOOLEAN
				FROM
					data d
				WHERE NOT EXISTS (
					SELECT 1 FROM user_emails ue
					WHERE LOWER(ue.email) = LOWER(d.email)
				)
			RETURNING
				(SELECT user_id FROM new_user);
	`,

	markUserAsDeleted: `
		UPDATE users SET deleted_at = NOW() AT TIME ZONE 'UTC' WHERE user_id = $1;
	`,

	markUserAppsAsDeleted: `
		UPDATE apps SET deleted_at = NOW() AT TIME ZONE 'UTC' WHERE user_id = $1 RETURNING app_id;
	`,

	updateSubscription: `
		UPDATE
			users
		SET
			metadata = $1,
			updated_at = NOW() AT TIME ZONE 'UTC'
		WHERE
			user_id = $2;
	`,

	updatePersonalAccessToken: `
		UPDATE
			user_access_tokens
		SET
			personal_access_token = $1
		WHERE
			user_id = $2;
	`,

	selectLicense: `
		SELECT
			license_key, license_version, number_of_seats, user_id
		FROM
			licenses
		WHERE
			{{ .where }};
	`,

	insertLicense: `
		INSERT INTO licenses
			(license_key, license_version, number_of_seats, user_id, metadata)
		VALUES
			($1, $2, $3, $4, $5);
	`,

	deleteLicense: `
		DELETE FROM licenses WHERE user_id = $1;
	`,

	updateLastLogin: `
		UPDATE users SET last_login_at = NOW() WHERE user_id = $1;
	`,

	userMetrics: `
		SELECT
			um.bandwidth_bytes, um.function_invocations,
			um.storage_bytes, um.build_minutes,
			u.user_id, u.metadata
		FROM
			user_metrics um
		JOIN
			users u ON u.user_id = um.user_id
		WHERE
			um.user_id = {{ .where }} AND
			um.year = EXTRACT(YEAR FROM CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::INTEGER AND
			um.month = EXTRACT(MONTH FROM CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::INTEGER;
	`,

	updateUsageMetrics: `
		INSERT INTO user_metrics (
			user_id,
			bandwidth_bytes,
			function_invocations,
			year,
			month
		)
		VALUES
			{{ generateValues 5 (len .) }}
		ON CONFLICT (user_id, year, month)
		DO UPDATE SET
			bandwidth_bytes = user_metrics.bandwidth_bytes + EXCLUDED.bandwidth_bytes,
			function_invocations = user_metrics.function_invocations + EXCLUDED.function_invocations;
	`,
}
