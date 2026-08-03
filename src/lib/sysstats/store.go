package sysstats

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
)

const (
	// Retention is how far back the history goes. Deliberately short: this is a
	// health view, not a metrics store. Longer horizons belong in a real
	// Prometheus scraping the same exporters.
	Retention = 24 * time.Hour

	// keyTTL outlives Retention so a machine that stops reporting takes its
	// history with it instead of leaking a key forever.
	keyTTL = 25 * time.Hour

	historyPrefix = "metrics:history:"
)

// Store keeps per-target sample history in a Redis sorted set scored by
// timestamp.
type Store struct {
	client    *rediscache.RedisCache
	retention time.Duration
}

type NewStoreParams struct {
	Client *rediscache.RedisCache

	// Retention overrides the default window. Tests use it to avoid waiting a
	// day to observe trimming.
	Retention time.Duration
}

func NewStore(p NewStoreParams) *Store {
	client := p.Client

	if client == nil {
		client = rediscache.Client()
	}

	retention := p.Retention

	if retention == 0 {
		retention = Retention
	}

	return &Store{client: client, retention: retention}
}

func historyKey(target string) string {
	return historyPrefix + target
}

// Append writes a sample and trims anything older than the retention window.
func (s *Store) Append(ctx context.Context, sample *Sample) error {
	if s.client == nil {
		return fmt.Errorf("redis is not available")
	}

	if sample.Target == "" {
		return fmt.Errorf("sample has no target")
	}

	payload, err := json.Marshal(sample)

	if err != nil {
		return err
	}

	key := historyKey(sample.Target)
	cutoff := time.Unix(sample.Timestamp, 0).Add(-s.retention).Unix()

	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(sample.Timestamp), Member: payload})
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("(%d", cutoff))
	pipe.Expire(ctx, key, keyTTL)

	_, err = pipe.Exec(ctx)

	return err
}

// Read returns samples recorded at or after since, oldest first.
func (s *Store) Read(ctx context.Context, target string, since time.Time) ([]*Sample, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis is not available")
	}

	raw, err := s.client.ZRangeByScore(ctx, historyKey(target), &redis.ZRangeBy{
		Min: strconv.FormatInt(since.Unix(), 10),
		Max: "+inf",
	}).Result()

	if err != nil {
		return nil, err
	}

	return decodeSamples(raw), nil
}

// Latest returns the most recent sample for a target, or nil when none has been
// recorded yet.
func (s *Store) Latest(ctx context.Context, target string) (*Sample, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis is not available")
	}

	raw, err := s.client.ZRevRangeByScore(ctx, historyKey(target), &redis.ZRangeBy{
		Min:   "-inf",
		Max:   "+inf",
		Count: 1,
	}).Result()

	if err != nil {
		return nil, err
	}

	samples := decodeSamples(raw)

	if len(samples) == 0 {
		return nil, nil
	}

	return samples[0], nil
}

// Drop removes a target's history. Used when a machine is removed from the
// manual target list, so it stops appearing in the UI immediately rather than
// lingering until its key expires.
func (s *Store) Drop(ctx context.Context, target string) error {
	if s.client == nil {
		return fmt.Errorf("redis is not available")
	}

	return s.client.Del(ctx, historyKey(target)).Err()
}

// decodeSamples skips entries it cannot parse. A sample written by an older
// build with an incompatible shape should not take the whole series down.
func decodeSamples(raw []string) []*Sample {
	out := make([]*Sample, 0, len(raw))

	for _, item := range raw {
		var sample Sample

		if err := json.Unmarshal([]byte(item), &sample); err != nil {
			continue
		}

		out = append(out, &sample)
	}

	return out
}
