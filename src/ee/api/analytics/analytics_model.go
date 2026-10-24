package analytics

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

type Record struct {
	ID          types.ID
	DomainID    types.ID
	AppID       types.ID
	EnvID       types.ID
	VisitorIP   string
	RequestPath string
	HostName    string
	Referrer    null.String
	UserAgent   null.String
	RequestTS   utils.Unix
	StatusCode  int
	RequestID   null.String
	// Source distinguishes server-recorded hits ("server") from client beacon
	// pageviews ("client"). NULL (legacy rows) is treated as server.
	Source null.String
	// Campaign is read from the landing URL's query.
	Campaign Campaign
}

func (r Record) String() string {
	referrer := "<null>"
	if r.Referrer.Valid {
		referrer = r.Referrer.String
	}

	userAgent := "<null>"
	if r.UserAgent.Valid {
		userAgent = r.UserAgent.String
	}

	return fmt.Sprintf("Record(ID: %d, DomainID: %d, AppID: %d, EnvID: %d, VisitorIP: %s, RequestPath: %s, HostName: %s, Referrer: %s, UserAgent: %s, RequestTS: %s, StatusCode: %d) \n",
		r.ID, r.DomainID, r.AppID, r.EnvID, r.VisitorIP, r.RequestPath, r.HostName, referrer, userAgent, r.RequestTS, r.StatusCode)
}

// androidAppScheme prefixes referrers sent by Android apps, e.g.
// android-app://com.google.android.googlequicksearchbox/https/www.google.com.
const androidAppScheme = "android-app://"

// NormalizeReferrer reduces a referrer to its host name. An Android app
// referrer has no host, so its package name stands in for it.
func NormalizeReferrer(referrer string) string {
	if referrer == "" {
		return ""
	}

	if rest, ok := strings.CutPrefix(strings.ToLower(referrer), androidAppScheme); ok {
		pkg, _, _ := strings.Cut(rest, "/")

		return pkg
	}

	if !strings.HasPrefix(referrer, "http") {
		referrer = "https://" + referrer
	}

	// Get the hostname from the referrer
	parsed, _ := url.Parse(referrer)

	if parsed == nil {
		return ""
	}

	return parsed.Hostname()
}
