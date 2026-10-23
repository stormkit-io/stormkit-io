package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
)

// VisitorIDParams holds the inputs to VisitorID. The struct keeps the two
// same-typed strings from being transposed at a call site.
type VisitorIDParams struct {
	IP        string
	UserAgent string

	// At is when the visit happened; it picks the day the id rotates on.
	// Zero means now.
	At time.Time
}

// VisitorID returns a cookieless, daily-rotating identifier for a visitor.
// It is derived from a server secret, the current UTC day, the visitor IP and
// the user agent, so the same visitor maps to a stable id within a day without
// persisting any PII. The secret and daily rotation make the hash impractical
// to reverse or correlate across days.
func VisitorID(p VisitorIDParams) string {
	at := p.At

	if at.IsZero() {
		at = time.Now()
	}

	day := at.UTC().Format(time.DateOnly)
	seed := config.Get().AppSecret + "|" + day + "|" + p.IP + "|" + p.UserAgent
	sum := sha256.Sum256([]byte(seed))

	return hex.EncodeToString(sum[:16])
}
