package buildconf

import (
	"encoding/json"

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

type Email struct {
	ID      types.ID   `json:"id,string"`
	EnvID   types.ID   `json:"envId,string"`
	From    string     `json:"from"`
	To      string     `json:"to"`
	Subject string     `json:"subject"`
	Body    string     `json:"body"`
	SentAt  utils.Unix `json:"sentAt"`
}
