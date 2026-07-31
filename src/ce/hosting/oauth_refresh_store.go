package hosting

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// errRefreshNotFound is returned when a refresh token is unknown, expired or
// already rotated, so the token endpoint can answer with RFC 6749 invalid_grant.
var errRefreshNotFound = errors.New("oauth refresh token not found")

// oauthRefreshTTL bounds how long a refresh token — and therefore an idle
// connection — survives without use. Each rotation issues a fresh token with a
// new window, so an actively used connector renews indefinitely while an
// abandoned one is reaped.
const oauthRefreshTTL = 30 * 24 * time.Hour

// oauthRefreshPayload is the state a refresh token stands in for. It carries the
// identity and grant parameters needed to mint a replacement access token
// without re-consent.
type oauthRefreshPayload struct {
	UID      string `json:"uid"`
	EML      string `json:"eml,omitempty"`
	ClientID string `json:"clientId"`
	Scope    string `json:"scope,omitempty"`
	Audience string `json:"aud,omitempty"`
}

// oauthRefreshStore is the Redis-backed rotating refresh-token registry. Unlike
// the 5-minute authorization codes, refresh tokens live for weeks, so the token
// itself is never stored: the key is its SHA-256 digest. A Redis dump therefore
// yields no replayable tokens — the endpoint only accepts a value whose digest
// is present, and the digest can't be reversed.
type oauthRefreshStore struct {
	prefix string
}

var oauthRefreshTokens = oauthRefreshStore{prefix: "oauth:refresh:"}

// key is the Redis key for token: prefix + base64url(sha256(token)).
//
// The encoding is part of the key format: changing it (padded vs unpadded,
// URL-safe vs standard) orphans every refresh token already in Redis.
func (s oauthRefreshStore) key(token string) string {
	sum := sha256.Sum256([]byte(token))

	return s.prefix + base64.RawURLEncoding.EncodeToString(sum[:])
}

// issue stashes payload under a fresh crypto-random refresh token and returns
// the token to hand back to the client.
func (s oauthRefreshStore) issue(ctx context.Context, payload oauthRefreshPayload) (string, error) {
	client := rediscache.Client()

	if client == nil {
		return "", errors.New("redis client is not available")
	}

	blob, err := json.Marshal(payload)

	if err != nil {
		return "", err
	}

	token, err := utils.SecureRandomToken(48)

	if err != nil {
		return "", err
	}

	if err := client.Set(ctx, s.key(token), blob, oauthRefreshTTL).Err(); err != nil {
		return "", err
	}

	return token, nil
}

// revoke deletes token from the store if present. It is idempotent — an unknown,
// expired or already-rotated token is a no-op — which matches RFC 7009's rule
// that revocation report success regardless, so a caller can't probe validity.
func (s oauthRefreshStore) revoke(ctx context.Context, token string) error {
	client := rediscache.Client()

	if client == nil {
		return errors.New("redis client is not available")
	}

	return client.Del(ctx, s.key(token)).Err()
}

// rotate atomically consumes token and returns its payload. Rotation is
// mandatory for public clients (OAuth 2.1 / the MCP auth spec): the presented
// token is deleted in the same round-trip, so a captured-then-rotated token is
// dead on arrival. It returns errRefreshNotFound for an unknown, expired or
// already-rotated token and any other error verbatim.
func (s oauthRefreshStore) rotate(ctx context.Context, token string) (oauthRefreshPayload, error) {
	client := rediscache.Client()

	if client == nil {
		return oauthRefreshPayload{}, errors.New("redis client is not available")
	}

	raw, err := client.GetDel(ctx, s.key(token)).Result()

	if errors.Is(err, redis.Nil) || raw == "" {
		return oauthRefreshPayload{}, errRefreshNotFound
	}

	if err != nil {
		return oauthRefreshPayload{}, err
	}

	var payload oauthRefreshPayload

	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return oauthRefreshPayload{}, err
	}

	return payload, nil
}
