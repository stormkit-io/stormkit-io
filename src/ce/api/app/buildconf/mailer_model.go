package buildconf

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// SendMailFunc is the underlying SMTP send function, replaceable in tests.
var SendMailFunc = SendMailWithDeadline

// sendTimeout bounds a single delivery attempt. It stays comfortably below the
// server's handler timeout (STORMKIT_HTTP_READ_TIMEOUT, 30s by default) so that
// a stalled relay surfaces as an SMTP error rather than as a generic gateway
// timeout the caller cannot act on.
const sendTimeout = 10 * time.Second

// SendMailWithDeadline is smtp.SendMail with a dial and I/O deadline. The
// stdlib version sets none, so an unreachable or black-holing host would hold
// the connection - and the credentials and message it carries - until the OS
// TCP timeout. The deadline aborts the attempt itself, so nothing outlives the
// call and a returned error means the message really was not handed over.
func SendMailWithDeadline(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	host, _, err := net.SplitHostPort(addr)

	if err != nil {
		return err
	}

	deadline := time.Now().Add(sendTimeout)
	conn, err := (&net.Dialer{Deadline: deadline}).Dial("tcp", addr)

	if err != nil {
		return err
	}

	defer conn.Close()

	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, host)

	if err != nil {
		return err
	}

	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return err
			}
		}
	}

	if err := client.Mail(from); err != nil {
		return err
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := client.Data()

	if err != nil {
		return err
	}

	if _, err := writer.Write(msg); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

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

	// UseDefault makes the environment send through the instance's mailer,
	// configured by the instance admin, instead of SMTP settings of its own.
	UseDefault bool `json:"useDefault,omitempty"`

	// DefaultFrom is the From address of emails that do not set one. The
	// instance mailer sets it to the address the admin configured.
	DefaultFrom string `json:"-"`
}

// mailerConfStorage is how a MailerConf is persisted. Every value is a string
// because versions before UseDefault decode the column into a string map, and
// a boolean would make them fail to load the environment after a rollback.
type mailerConfStorage struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	UseDefault string `json:"useDefault,omitempty"`
}

// Bytes returns the representation persisted in the database, with the
// encrypted credentials rather than the placeholder rendered for clients.
func (mc *MailerConf) Bytes() ([]byte, error) {
	cnf := mailerConfStorage{
		Host:     mc.Host,
		Port:     mc.Port,
		Username: utils.EncryptToString(mc.Username),
		Password: utils.EncryptToString(mc.Password),
	}

	if mc.UseDefault {
		cnf.UseDefault = "true"
	}

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
		"host":       mc.Host,
		"port":       mc.Port,
		"username":   mc.Username,
		"password":   password,
		"useDefault": mc.UseDefault,
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

	stored := mailerConfStorage{}

	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}

	if mc == nil {
		mc = &MailerConf{}
	}

	mc.Username = utils.DecryptToString(stored.Username)
	mc.Password = utils.DecryptToString(stored.Password)
	mc.Host = stored.Host
	mc.Port = stored.Port
	mc.UseDefault = stored.UseDefault == "true"

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
	fromHeader := SanitizeHeader(mc.From(p.From))
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

// From returns the From header of an email the caller wants sent from
// requested: the requested address, the default one, or the username.
func (mc *MailerConf) From(requested string) string {
	return utils.GetString(requested, mc.DefaultFrom, mc.Username)
}

// Mailer returns the SMTP settings env sends emails with: its own, the
// instance mailer when it opted into that one, or nil when it has neither.
func (env *Env) Mailer() *MailerConf {
	if env == nil || env.MailerConf == nil {
		return nil
	}

	if !env.MailerConf.UseDefault {
		return env.MailerConf
	}

	return InstanceMailer()
}

// InstanceMailer returns the instance's default mailer, or nil when the admin
// has not configured one. It is a self-hosted feature, so it is always nil on
// Stormkit Cloud.
func InstanceMailer() *MailerConf {
	if config.IsStormkitCloud() {
		return nil
	}

	cnf := admin.MustConfig().MailerConfig

	if !cnf.IsConfigured() {
		return nil
	}

	return &MailerConf{
		Host:        cnf.Host,
		Port:        cnf.Port,
		Username:    cnf.Username,
		Password:    cnf.Password,
		DefaultFrom: cnf.From,
	}
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
