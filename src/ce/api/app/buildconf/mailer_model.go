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
	From    string // falls back to mc.DefaultFrom() when empty
	Subject string
	Body    string
}

type MailerConf struct {
	EnvID       types.ID `json:"-"`
	Host        string   `json:"host"`
	Port        string   `json:"port"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	FromAddress string   `json:"fromAddress,omitempty"`
}

// Bytes uses the json marshaler to return the byte representation.
func (mc *MailerConf) Bytes() ([]byte, error) {
	cnf := *mc
	cnf.Username = utils.EncryptToString(cnf.Username)
	cnf.Password = utils.EncryptToString(cnf.Password)
	return json.Marshal(cnf)
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
	mc.FromAddress = hash["fromAddress"]

	return nil
}

// DefaultFrom returns the address used as the From header when a send
// request does not specify one: the configured FromAddress, falling back
// to the SMTP username.
func (mc *MailerConf) DefaultFrom() string {
	return utils.GetString(mc.FromAddress, mc.Username)
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
	fromHeader := SanitizeHeader(utils.GetString(p.From, mc.DefaultFrom()))
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
