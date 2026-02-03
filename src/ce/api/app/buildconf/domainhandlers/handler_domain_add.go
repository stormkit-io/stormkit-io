package domainhandlers

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

const urlRegex = `^(https?:\/\/)?(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,63}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)$`

type addDomainRequest struct {
	// Domain is the name of the domain that will point to Env.
	Domain string `json:"domain"`
}

// IsValidDomain checks whether the given domain is valid or not.
// If the URL is valid, this method returns a *url.URL instance,
// otherwise nil.
func IsValidDomain(domain string) *url.URL {
	// Ensure the domain has a protocol; if not, add "http://"
	if !strings.Contains(domain, "://") {
		domain = "http://" + domain
	}

	// Parse the URL
	parsedUrl, err := url.Parse(domain)

	if err != nil || parsedUrl.Host == "" {
		return nil
	}

	if regexp.MustCompile(urlRegex).MatchString(domain) {
		return parsedUrl
	}

	return nil
}

// HandlerDomainAdd sets the domain for the given environment.
func HandlerDomainAdd(req *app.RequestContext) *shttp.Response {
	sdr := &addDomainRequest{}

	if err := req.Post(sdr); err != nil {
		return shttp.Error(err)
	}

	sdr.Domain = strings.TrimSpace(strings.ToLower(sdr.Domain))

	parsed := IsValidDomain(sdr.Domain)

	if parsed == nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Please provide a valid domain name.",
			},
		}
	}

	domain := &buildconf.DomainModel{
		AppID: req.App.ID,
		EnvID: req.EnvID,
		Name:  parsed.Hostname(),
		Token: null.NewString(utils.RandomToken(32), true),
	}

	isSelfHosted := config.IsSelfHosted()

	if isSelfHosted {
		domain.Verified = true
		domain.VerifiedAt = utils.NewUnix()
	}

	existing, err := buildconf.DomainStore().DomainByName(req.Context(), domain.Name)

	if err != nil {
		return shttp.Error(err)
	}

	if existing != nil && existing.Verified {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "This domain is already in use.",
			},
		}
	}

	if err := buildconf.DomainStore().Insert(req.Context(), domain); err != nil {
		return shttp.Error(err)
	}

	// Reset cache in case there is a 404 for the domain in the cache.
	// We only do this on self hosted instances because domain verification is instant.
	if isSelfHosted {
		if err := appcache.Service().Reset(0, domain.Name); err != nil {
			slog.Errorf("failed resetting cache after domain insert: %s", err.Error())
		}
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.CreateAction, audit.TypeDomain).
			WithDiff(&audit.Diff{New: audit.DiffFields{DomainName: domain.Name}}).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"domainId": domain.ID.String(),
			"token":    domain.Token.ValueOrZero(),
		},
	}
}
