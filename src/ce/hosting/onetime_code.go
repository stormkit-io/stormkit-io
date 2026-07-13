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

// errCodeNotFound is returned by oneTimeCode.redeem when the code is absent or
// already consumed, letting callers distinguish "unknown code" from a Redis
// error.
var errCodeNotFound = errors.New("one-time code not found")

// oneTimeCode is a Redis-backed single-use code store. It marshals a JSON
// payload under a freshly generated, crypto-random code with a TTL, then reads
// it back once via GetDel (atomic single-use). Both the OAuth authorization
// codes and the magic-link session codes are built on it; the prefix keeps
// their keyspaces from colliding.
type oneTimeCode struct {
	prefix string
	ttl    time.Duration
}

// issue stashes payload under a fresh code and returns the code.
func (c oneTimeCode) issue(ctx context.Context, payload any) (string, error) {
	blob, err := json.Marshal(payload)

	if err != nil {
		return "", err
	}

	code, err := utils.SecureRandomToken(48)

	if err != nil {
		return "", err
	}

	if err := rediscache.Client().Set(ctx, c.prefix+code, blob, c.ttl).Err(); err != nil {
		return "", err
	}

	return code, nil
}

// redeem atomically reads-and-deletes the payload for code into dst. It returns
// errCodeNotFound when the code is unknown or already used, and any other error
// verbatim so callers can distinguish a missing code from a store failure.
func (c oneTimeCode) redeem(ctx context.Context, code string, dst any) error {
	raw, err := rediscache.Client().GetDel(ctx, c.prefix+code).Result()

	if errors.Is(err, redis.Nil) || raw == "" {
		return errCodeNotFound
	}

	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(raw), dst)
}
