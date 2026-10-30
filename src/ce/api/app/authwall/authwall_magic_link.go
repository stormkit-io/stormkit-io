package authwall

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// MagicLinkMaxMins is how long a sign-in link sent by email works.
const MagicLinkMaxMins = 15

// MagicLink is a sign-in link sent by email.
type MagicLink struct {
	EnvID   types.ID `json:"envId"`
	LoginID types.ID `json:"loginId"`

	// Email is the email the link was sent to.
	Email string `json:"email"`

	// ReturnTo is the protected page the visitor asked for the link on.
	ReturnTo string `json:"returnTo"`
}

// MagicLinks keeps sign-in links until they are used or expire. A link's URL
// carries only a random ID; what it signs in to stays here, so the email holds
// nothing a visitor submitted, and using a link deletes it.
type MagicLinks struct{}

func (MagicLinks) key(id string) string {
	return "authwall:magic:" + id
}

// Create stores the link and returns its ID.
func (m MagicLinks) Create(ctx context.Context, link MagicLink) (string, error) {
	id, err := utils.SecureRandomToken(32)

	if err != nil {
		return "", err
	}

	value, err := json.Marshal(link)

	if err != nil {
		return "", err
	}

	if err := rediscache.Client().Set(ctx, m.key(id), value, MagicLinkMaxMins*time.Minute).Err(); err != nil {
		return "", err
	}

	return id, nil
}

// Get returns the link with the ID, or nil when there is none. The link stays
// usable.
func (m MagicLinks) Get(ctx context.Context, id string) (*MagicLink, error) {
	if id == "" {
		return nil, nil
	}

	value, err := rediscache.Client().Get(ctx, m.key(id)).Bytes()

	if errors.Is(err, redis.Nil) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	link := &MagicLink{}

	if err := json.Unmarshal(value, link); err != nil {
		return nil, err
	}

	return link, nil
}

// Use returns the link with the ID and deletes it, or returns nil when there
// is none. Of two visitors using the same link at once, only one gets it.
func (m MagicLinks) Use(ctx context.Context, id string) (*MagicLink, error) {
	link, err := m.Get(ctx, id)

	if err != nil || link == nil {
		return nil, err
	}

	deleted, err := rediscache.Client().Del(ctx, m.key(id)).Result()

	if err != nil {
		return nil, err
	}

	if deleted == 0 {
		return nil, nil
	}

	return link, nil
}
