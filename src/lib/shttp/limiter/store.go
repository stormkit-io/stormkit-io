package limiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Keep a copy of created stores for the cleanup function.
var stores = []*Store{}

// Visit represents a user visit.
type Visit struct {
	Limiter  *rate.Limiter
	Count    int64
	LastSeen time.Time
}

// Store is an in-memory store for handling rate limits.
type Store struct {
	// Visits is a map of ip-visit pairs. On each request it is
	// filled with records based on request ip. These records are
	// cleaned up by the Cleanup method.
	Visits map[string]*Visit

	// limit is the number of requests a given ip can perform during
	// the given duration.
	Limit int64

	// The number of maximum credits which the user has before consuming
	// all tokens. The burst tokens are refilled every 1 / limit second.
	Burst int

	// duration is the amount of time which the limit will be applied.
	Duration time.Duration

	// Hash is an array of strings which specifies the hashes that will need to be
	// included in the request. By default it is ip and path.
	Hash []string

	mtx sync.Mutex
}

// NewStore creates a new store instance.
// Limit is the number of events that is allowed for a given user
// during the given 'duration'. For instance, 5, 60 * time.Second
// would allow 5 events per minute, per user.
func NewStore(opts *Options) *Store {
	if opts == nil {
		opts = &Options{}
	}

	if opts.Limit == 0 {
		opts.Limit = 10
	}

	if opts.Duration == 0 {
		opts.Duration = time.Minute
	}

	if opts.Burst == 0 {
		opts.Burst = 10
	}

	if len(opts.Hash) == 0 {
		opts.Hash = []string{"ip", "path"}
	}

	store := &Store{
		Visits:   make(map[string]*Visit),
		Hash:     opts.Hash,
		Limit:    opts.Limit,
		Duration: opts.Duration,
		Burst:    opts.Burst,
	}

	stores = append(stores, store)
	return store
}

// Add creates a new rate limiter and add it to the Visitors map, using the
// IP address as the key.
func (s *Store) Add(ip string) *Visit {
	s.mtx.Lock()
	eventsPerSecond := float64(s.Limit) / s.Duration.Seconds()

	s.Visits[ip] = &Visit{
		Limiter:  rate.NewLimiter(rate.Limit(eventsPerSecond), s.Burst),
		LastSeen: time.Now(),
		Count:    1,
	}

	s.mtx.Unlock()
	return s.Visits[ip]
}

// Get retrieves and returns the rate limiter for the current Visitor if it
// already exists. Otherwise call the add function to add a new entry to the map.
func (s *Store) Get(hash string) *Visit {
	s.mtx.Lock()
	visit, exists := s.Visits[hash]

	if !exists {
		s.mtx.Unlock()
		return s.Add(hash)
	}

	visit.LastSeen = time.Now()
	visit.Count = visit.Count + 1
	s.mtx.Unlock()
	return visit
}

// Cleanup checks every minute the map for Visitors that haven't been seen for
// more than `duration` minutes and delete the entries.
func Cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C

		for _, s := range stores {
			s.mtx.Lock()

			for ip, v := range s.Visits {
				if time.Since(v.LastSeen) > s.Duration {
					delete(s.Visits, ip)
				}
			}

			s.mtx.Unlock()
		}
	}
}
