package appcache_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type CacheSuite struct {
	suite.Suite
	*factory.Factory

	domains []*buildconf.DomainModel
	user    *factory.MockUser
	app     *factory.MockApp
	env     *factory.MockEnv
	ctx     context.Context

	conn databasetest.TestDB
}

func (s *CacheSuite) SetupSuite() {
	s.conn = databasetest.InitTx("appcache_suite")
	s.Factory = factory.New(s.conn)

	s.ctx = context.Background()
	s.user = s.MockUser()
	s.app = s.MockApp(s.user)
	s.env = s.MockEnv(s.app)

	s.domains = []*buildconf.DomainModel{
		{
			AppID:      s.app.ID,
			EnvID:      s.env.ID,
			Name:       "example.org",
			Verified:   true,
			VerifiedAt: utils.NewUnix(),
		},
		{
			AppID:      s.app.ID,
			EnvID:      s.env.ID,
			Name:       "www.example.org",
			Verified:   true,
			VerifiedAt: utils.NewUnix(),
		},
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), s.domains[0]))
	s.NoError(buildconf.DomainStore().Insert(context.Background(), s.domains[1]))
}

func (s *CacheSuite) Test_ResetCache() {
	service := rediscache.Service()

	var mu sync.Mutex
	msgs := []string{}

	snapshot := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(msgs)
	}

	s.NoError(service.SubscribeAsync(rediscache.EventInvalidateHostingCache, func(ctx context.Context, payload ...string) {
		mu.Lock()
		defer mu.Unlock()
		msgs = append(msgs, payload...)
	}))

	s.NoError(appcache.Service().Reset(s.env.ID))

	firstBatch := []string{
		"example.org",
		fmt.Sprintf(`^%s(?:--\d+)?`, s.app.DisplayName),
		"www.example.org",
	}

	sortedEqual := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		ac, bc := slices.Clone(a), slices.Clone(b)
		slices.Sort(ac)
		slices.Sort(bc)
		return slices.Equal(ac, bc)
	}

	// Wait for the first batch to arrive before issuing the second Reset,
	// so messages from both calls don't interleave.
	// ResetCacheArgs has no ORDER BY, so compare order-insensitively.
	s.Require().Eventually(func() bool {
		return sortedEqual(firstBatch, snapshot())
	}, 5*time.Second, 100*time.Millisecond)

	// With filters
	s.NoError(appcache.Service().Reset(0, "www.example.org"))

	s.Eventually(func() bool {
		snap := snapshot()
		// The last message is always deterministic (direct key from second Reset).
		// The first 3 can arrive in any order due to no ORDER BY in ResetCacheArgs.
		return len(snap) == 4 &&
			snap[len(snap)-1] == "www.example.org" &&
			sortedEqual(snap[:3], firstBatch)
	}, 5*time.Second, 100*time.Millisecond)
}

func TestCacheSuite(t *testing.T) {
	suite.Run(t, &CacheSuite{})
}
