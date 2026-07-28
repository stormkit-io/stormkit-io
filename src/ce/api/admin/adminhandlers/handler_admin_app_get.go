package adminhandlers

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// volumeFilePathPrefix is the URL path prefix used by public volume file links
// (see volumes.File.PublicLink).
const volumeFilePathPrefix = "/volumes/file/"

// handlerAdminAppGet returns the app information, with the associated user.
func handlerAdminAppGet(req *user.RequestContext) *shttp.Response {
	addrs := defangURL(req.Query().Get("url"))
	store := app.NewStore()

	if !strings.HasPrefix(addrs, "http") && !strings.HasPrefix(addrs, "//") {
		addrs = fmt.Sprintf("https://%s", addrs)
	}

	parsed, err := url.Parse(addrs)

	if err != nil {
		return shttp.BadRequest(map[string]any{
			"error": "Invalid URL format provided",
		})
	}

	appl, err := resolveApp(req.Context(), store, addrs, parsed)

	if err != nil {
		return shttp.Error(err)
	}

	if appl == nil {
		return shttp.NoContent()
	}

	usr, err := user.NewStore().UserByID(appl.UserID)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: map[string]any{
			"app":  appl.JSON(),
			"user": usr.JSON(),
		},
	}
}

// defangURL reverses the common "defanging" applied to URLs in abuse and threat
// reports (e.g. AWS trust reports) so they can be parsed again: hxxp(s):// back
// to http(s):// and [.] back to a literal dot.
func defangURL(addrs string) string {
	replacer := strings.NewReplacer(
		"[.]", ".",
		"hxxps://", "https://",
		"hxxp://", "http://",
	)

	return replacer.Replace(addrs)
}

// resolveApp finds the app a URL points to. It first tries to resolve a public
// volume file URL (which embeds the environment ID) and otherwise falls back to
// hostname-based resolution (stormkit.dev display name or custom domain).
func resolveApp(ctx context.Context, store *app.Store, addrs string, parsed *url.URL) (*app.App, error) {
	if appl, err := appByVolumeURL(ctx, store, parsed); err != nil || appl != nil {
		return appl, err
	}

	hostname := parsed.Hostname()

	if appconf.IsStormkitDev(addrs) {
		return store.AppByDisplayName(ctx, strings.Split(hostname, ".")[0])
	}

	return store.AppByDomainName(ctx, hostname)
}

// appByVolumeURL resolves the app owning a public volume file URL of the form
// /volumes/file/<token>, where <token> is the encrypted "<fileID>:<envID>" pair
// produced by volumes.File.PublicLink. It returns (nil, nil) when the URL is not
// a volume file URL or its token cannot be decoded, so the caller falls back to
// hostname-based resolution.
func appByVolumeURL(ctx context.Context, store *app.Store, parsed *url.URL) (*app.App, error) {
	if !strings.HasPrefix(parsed.Path, volumeFilePathPrefix) {
		return nil, nil
	}

	token := strings.TrimPrefix(parsed.Path, volumeFilePathPrefix)
	decrypted := utils.DecryptToString(token)

	if decrypted == "" {
		return nil, nil
	}

	pieces := strings.Split(decrypted, ":")

	if len(pieces) != 2 {
		return nil, nil
	}

	return store.AppByEnvID(ctx, utils.StringToID(pieces[1]))
}
