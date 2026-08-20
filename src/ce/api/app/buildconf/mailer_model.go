package buildconf

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// SendMailFunc is the underlying SMTP send function, replaceable in tests.
var SendMailFunc = smtp.SendMail

type SendEmailParams struct {
	To      string // semicolon-separated list of recipients
	From    string // falls back to mc.Username when empty
	Subject string
	Body    string
}

// PasswordPlaceholder stands in for the stored SMTP password whenever a mailer
// configuration is rendered for a client. Writing it back leaves the stored
// password untouched, so a client can round-trip a configuration without ever
// handling the real secret.
const PasswordPlaceholder = "****-****-****-****"

type MailerConf struct {
	EnvID    types.ID `json:"-"`
	Host     string   `json:"host"`
	Port     string   `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
}

// Bytes uses the json marshaler to return the byte representation. It marshals
// a storage type that sheds MarshalJSON, so persistence keeps writing the
// encrypted credential rather than the placeholder rendered for clients.
func (mc *MailerConf) Bytes() ([]byte, error) {
	type mailerConfStorage MailerConf

	cnf := mailerConfStorage(*mc)
	cnf.Username = utils.EncryptToString(cnf.Username)
	cnf.Password = utils.EncryptToString(cnf.Password)

	return json.Marshal(cnf)
}

// MarshalJSON implements json.Marshaler by delegating to JSON, so masking is
// the default: a MailerConf placed in a response cannot serialize the password
// even if the caller forgets to call JSON explicitly.
func (mc *MailerConf) MarshalJSON() ([]byte, error) {
	return json.Marshal(mc.JSON())
}

// JSON renders the configuration for API responses. The password is never
// included; the placeholder marks that one is stored. Every client-facing
// serialization goes through here so the credential cannot escape by accident.
func (mc *MailerConf) JSON() map[string]any {
	if mc == nil {
		return nil
	}

	password := ""

	if mc.Password != "" {
		password = PasswordPlaceholder
	}

	return map[string]any{
		"host":     mc.Host,
		"port":     mc.Port,
		"username": mc.Username,
		"password": password,
	}
}

// String returns a connection string.
func (mc *MailerConf) String() string {
	port := mc.Port

	if port == "" {
		port = "587"
	}

	u := &url.URL{
		Scheme: "smtp",
		User:   url.UserPassword(mc.Username, mc.Password),
		Host:   mc.Host + ":" + port,
	}

	return u.String()
}

// UnmarshalJSON implements the marshaler interface.
func (mc *MailerConf) UnmarshalJSON(data []byte) error {
	if data == nil {
		return nil
	}

	hash := map[string]string{}

	if err := json.Unmarshal(data, &hash); err != nil {
		return err
	}

	if mc == nil {
		mc = &MailerConf{}
	}

	mc.Username = utils.DecryptToString(hash["username"])
	mc.Password = utils.DecryptToString(hash["password"])
	mc.Host = hash["host"]
	mc.Port = hash["port"]

	return nil
}

// SanitizeHeader strips carriage returns and newlines from an SMTP header value
// to prevent header injection attacks.
func SanitizeHeader(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(v)
}

// Send delivers an HTML email using the environment's SMTP configuration.
// p.From sets the From: header (may include a display name); the SMTP
// envelope MAIL FROM is the bare address parsed from it — display-name
// forms like `Acme <foo@bar.com>` are rejected by many SMTP servers with
// a 501.
func (mc *MailerConf) Send(p SendEmailParams) error {
	fromHeader := SanitizeHeader(utils.GetString(p.From, mc.Username))
	port := utils.GetString(mc.Port, "587")
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg := []byte(
		fmt.Sprintf("From: %s\n", fromHeader) +
			fmt.Sprintf("Subject: %s\n", SanitizeHeader(p.Subject)) +
			mime +
			fmt.Sprintf("<html><body>%s</body></html>", p.Body),
	)

	auth := smtp.PlainAuth("", mc.Username, mc.Password, mc.Host)

	to := strings.Split(p.To, ";")

	for i := range to {
		to[i] = strings.TrimSpace(to[i])
	}

	envelope := fromHeader

	if addr, err := mail.ParseAddress(fromHeader); err == nil {
		envelope = addr.Address
	}

	return SendMailFunc(mc.Host+":"+port, auth, envelope, to, msg)
}

type Email struct {
	ID      types.ID   `json:"id,string"`
	EnvID   types.ID   `json:"envId,string"`
	From    string     `json:"from"`
	To      string     `json:"to"`
	Subject string     `json:"subject"`
	Body    string     `json:"body"`
	SentAt  utils.Unix `json:"sentAt"`
}
