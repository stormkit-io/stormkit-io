package buildconf

import (
	"encoding/json"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type MailerConf struct {
	EnvID    types.ID `json:"-"`
	Host     string   `json:"host"`
	Port     string   `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
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

	return nil
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
