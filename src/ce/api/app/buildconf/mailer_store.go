package buildconf

import (
	"context"
	"database/sql"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

var mstmt = struct {
	selectConfig string
	upsertConfig string
	selectEmails string
	insertEmail  string
}{
	selectConfig: `
		SELECT
			COALESCE(mailer_conf, '{}')
		FROM
			apps_build_conf
		WHERE
			env_id = $1 AND
			deleted_at IS NULL;
	`,

	upsertConfig: `
		UPDATE apps_build_conf SET mailer_conf = $1 WHERE env_id = $2;	
	`,

	selectEmails: `
		SELECT
			email_id, env_id, email_to, email_from,
			email_subject, email_body, created_at
		FROM
			mailer
		WHERE
			env_id = $1
		ORDER BY
			email_id DESC
		LIMIT
			100;
	`,

	insertEmail: `
		INSERT INTO mailer
			(env_id, email_to, email_from, email_subject, email_body)
		VALUES
			($1, $2, $3, $4, $5);
	`,
}

// Store represents a store for volume management.
type mailerStore struct {
	*database.Store
}

// Store returns a new store instance.
func MailerStore() *mailerStore {
	return &mailerStore{
		Store: database.NewStore(),
	}
}

// UpsertConfig creates or updates the volumes config.
func (s *mailerStore) UpsertConfig(ctx context.Context, cnf *MailerConf) error {
	data, err := cnf.Bytes()

	if err != nil {
		return err
	}

	_, err = s.Exec(ctx, mstmt.upsertConfig, data, cnf.EnvID)
	return err
}

// InsertMail inserts a sent email to the database. This is mostly for auditing.
func (s *mailerStore) InsertEmail(ctx context.Context, mail Email) error {
	_, err := s.Exec(ctx, mstmt.insertEmail, mail.EnvID, mail.To, mail.From, mail.Body, mail.Subject)
	return err
}

// Emails returns the last sent 100 emails.
func (s *mailerStore) Emails(ctx context.Context, envID types.ID) ([]*Email, error) {
	rows, err := s.Query(ctx, mstmt.selectEmails, envID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	defer rows.Close()

	emails := []*Email{}

	for rows.Next() {
		email := &Email{}
		err := rows.Scan(
			&email.ID, &email.EnvID, &email.To, &email.From,
			&email.Subject, &email.Body, &email.SentAt,
		)

		if err != nil {
			return nil, err
		}

		emails = append(emails, email)
	}

	return emails, nil
}
