package mailerhandlers

// RequestData is an outgoing email as submitted by the API.
type RequestData struct {
	To      string `json:"to"`
	From    string `json:"from"`
	Body    string `json:"body"`
	Subject string `json:"subject"`
}
