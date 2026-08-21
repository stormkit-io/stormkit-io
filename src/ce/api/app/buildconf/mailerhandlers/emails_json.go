package mailerhandlers

import (
	"strings"
	"unicode/utf8"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
)

// EmailsJSON renders the mailer log for the public API and the MCP tools.
//
// Bodies are withheld: the log stores magic-link emails verbatim, so a body
// contains a sign-in link for an end user of the app. The link is single-use
// and expires after 15 minutes, but delivery can be verified from the metadata
// alone, so there is no reason to put one in an agent transcript.
//
// Recipients are masked for the same audience. The full list is the customer's
// end-user mailing list, and an environment API key should not be able to walk
// off with it in one request. The domain survives so that a caller can still
// confirm a message reached the address it was aimed at. The dashboard renders
// the unmasked log through its own session-authenticated route.
func EmailsJSON(emails []*buildconf.Email) []map[string]any {
	out := make([]map[string]any, 0, len(emails))

	for _, e := range emails {
		out = append(out, map[string]any{
			"id":      e.ID.String(),
			"envId":   e.EnvID.String(),
			"from":    e.From,
			"to":      MaskRecipients(e.To),
			"subject": e.Subject,
			"sentAt":  e.SentAt,
		})
	}

	return out
}

// MaskRecipients masks the local part of every address in a recipient list,
// which the mailer stores semicolon-separated.
func MaskRecipients(to string) string {
	addresses := strings.Split(to, ";")

	for i, address := range addresses {
		addresses[i] = maskAddress(strings.TrimSpace(address))
	}

	return strings.Join(addresses, "; ")
}

// maskAddress turns jane@example.com into j***@example.com. An address without
// an "@" is masked whole rather than passed through, so a malformed value
// cannot leak by falling outside the rule.
func maskAddress(address string) string {
	if address == "" {
		return ""
	}

	at := strings.LastIndex(address, "@")

	if at < 1 {
		return "***"
	}

	// The first rune, not the first byte: slicing bytes splits a multi-byte
	// local part and the invalid UTF-8 surfaces as U+FFFD once serialized.
	_, size := utf8.DecodeRuneInString(address)

	return address[:size] + "***" + address[at:]
}
