package mailerhandlers

import (
	"context"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// SendValidationError is a user-facing validation failure on an outgoing
// email. Handlers surface Message in the 400 response body.
type SendValidationError struct {
	Message string
}

func (e *SendValidationError) Error() string { return e.Message }

// SendAndRecordParams carries the environment to send from and the message.
type SendAndRecordParams struct {
	Env  *buildconf.Env
	Data RequestData
}

// SendAndRecord appends data to the mailer log and, when the environment has
// an SMTP configuration, delivers it. It reports whether the message was
// actually handed to an SMTP server.
//
// An environment without a mailer records the email without sending it. That
// is deliberate — it lets an app use the mailer log before an SMTP server is
// configured — so callers that need to know delivery happened must consult the
// returned flag rather than a nil error.
func SendAndRecord(ctx context.Context, p SendAndRecordParams) (delivered bool, err error) {
	env := p.Env
	data := p.Data

	if strings.TrimSpace(data.Body) == "" {
		return false, &SendValidationError{Message: "Email body is a required field."}
	}

	if strings.TrimSpace(data.Subject) == "" {
		return false, &SendValidationError{Message: "Subject is a required field."}
	}

	from := data.From
	var sendErr error

	if env.MailerConf != nil {
		from = utils.GetString(data.From, env.MailerConf.Username)

		sendErr = env.MailerConf.Send(buildconf.SendEmailParams{
			To:      data.To,
			From:    data.From,
			Subject: data.Subject,
			Body:    data.Body,
		})

		delivered = sendErr == nil
	}

	// The attempt is recorded even when it failed. A relay can reject after it
	// has already accepted the message, so returning early would leave a
	// delivered email with no trace of who sent it or what it said.
	err = buildconf.MailerStore().InsertEmail(ctx, buildconf.Email{
		EnvID:   env.ID,
		From:    from,
		To:      data.To,
		Body:    data.Body,
		Subject: data.Subject,
	})

	if sendErr != nil {
		return false, sendErr
	}

	return delivered, err
}
