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

func NormalizeReferrer(referrer string) string {
	if referrer == "" {
		return ""
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
