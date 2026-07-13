package hosting

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// errClientNotFound is returned by oauthClientStore.get when the client_id is
// unknown or has expired, letting callers fall back to the origin-only check.
var errClientNotFound = errors.New("oauth client not found")

// oauthClientTTL is how long a registered client survives. It slides on each
// successful lookup, so an actively used connector never expires while abandoned
// registrations from the unauthenticated endpoint are reaped.
const oauthClientTTL = 30 * 24 * time.Hour

// registration rate-limit: a fixed window per host and caller IP blunts abuse
// of the unauthenticated Dynamic Client Registration endpoint.
const (
	registerRateMax    = 20
	registerRateWindow = time.Hour
)

// oauthClient is the persisted Dynamic Client Registration record (RFC 7591).
// Only the fields the authorization flow actually consumes are stored.
type oauthClient struct {
	ClientID     string   `json:"clientId"`
	ClientName   string   `json:"clientName,omitempty"`
	RedirectURIs []string `json:"redirectUris"`
	Scope        string   `json:"scope,omitempty"`
	IssuedAt     int64    `json:"issuedAt"`
}

// allowsRedirect reports whether redirectURI exactly matches one of the client's
// registered redirect_uris. Registration validates every URI against the
// operator's AllowedOrigins, so a registered match is strictly tighter than the
// origin-only check.
func (c oauthClient) allowsRedirect(redirectURI string) bool {
	for _, uri := range c.RedirectURIs {
		if uri == redirectURI {
			return true
		}
	}

	return false
}

// oauthClientStore is the Redis-backed Dynamic Client Registration registry. The
// "oauth:client:" prefix namespaces it away from the authorization codes.
type oauthClientStore struct {
	prefix string
}

var oauthClients = oauthClientStore{prefix: "oauth:client:"}

// register stashes a freshly-provisioned client under a generated client_id and
// returns the stored record (with ClientID populated).
func (s oauthClientStore) register(ctx context.Context, c oauthClient) (oauthClient, error) {
	client := rediscache.Client()

	if client == nil {
		return oauthClient{}, errors.New("redis client is not available")
	}

	id, err := utils.SecureRandomToken(24)

	if err != nil {
		return oauthClient{}, err
	}

	c.ClientID = id

	blob, err := json.Marshal(c)

	if err != nil {
		return oauthClient{}, err
	}

	if err := client.Set(ctx, s.prefix+id, blob, oauthClientTTL).Err(); err != nil {
		return oauthClient{}, err
	}

	return c, nil
}

// get reads a registered client and slides its TTL. It returns errClientNotFound
// for an unknown or expired client so callers can distinguish that from a store
// failure and fall back to the origin-only redirect check.
func (s oauthClientStore) get(ctx context.Context, clientID string) (oauthClient, error) {
	if clientID == "" {
		return oauthClient{}, errClientNotFound
	}

	client := rediscache.Client()

	if client == nil {
		return oauthClient{}, errClientNotFound
	}

	raw, err := client.GetEx(ctx, s.prefix+clientID, oauthClientTTL).Result()

	if errors.Is(err, redis.Nil) || raw == "" {
		return oauthClient{}, errClientNotFound
	}

	if err != nil {
		return oauthClient{}, err
	}

	var c oauthClient

	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return oauthClient{}, err
	}

	return c, nil
}

// registerAllowed enforces the fixed-window registration rate limit, keyed per
// host and caller IP so a single abusive caller can't burn the whole app's
// quota and lock legitimate connectors out of self-registration. It fails open
// on a Redis error: the register handler's own Set will surface a real outage
// as a server_error, so we don't want to double-reject on a transient blip.
func (s oauthClientStore) registerAllowed(ctx context.Context, host, ip string) bool {
	client := rediscache.Client()

	if client == nil {
		return true
	}

	key := "oauth:reg:rate:" + host + ":" + ip

	n, err := client.Incr(ctx, key).Result()

	if err != nil {
		return true
	}

	// Incr creates the key without a TTL, so arm one whenever the key has none.
	// Checking every call self-heals a counter left un-expiring by a failed
	// Expire or a crash between Incr and Expire; otherwise the key would live
	// forever and block this host+IP permanently once past the cap.
	if ttl, err := client.TTL(ctx, key).Result(); err == nil && ttl < 0 {
		client.Expire(ctx, key, registerRateWindow)
	}

	return n <= registerRateMax
}
