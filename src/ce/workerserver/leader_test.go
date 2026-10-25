package jobs_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stretchr/testify/suite"
)

type LeaderElectionSuite struct {
	suite.Suite

	key       string
	failRedis atomic.Bool
	failed    atomic.Int32
}

func (s *LeaderElectionSuite) SetupSuite() {
	s.key = "leader_election_test"
	rediscache.Client().AddHook(&failingRedisHook{suite: s})
}

func (s *LeaderElectionSuite) TearDownTest() {
	s.failRedis.Store(false)
	s.failed.Store(0)
}

// failingRedisHook fails leader checks and renewals on the suite's key while
// failRedis is set, simulating a Redis blip for the leader node only.
type failingRedisHook struct {
	suite *LeaderElectionSuite
}

func (h *failingRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *failingRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.suite.failRedis.Load() && h.touchesKey(cmd) {
			err := errors.New("simulated redis failure")
			cmd.SetErr(err)
			h.suite.failed.Add(1)

			return err
		}

		return next(ctx, cmd)
	}
}

func (h *failingRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

// touchesKey reports whether cmd is a leader check or renewal on the suite's key.
func (h *failingRedisHook) touchesKey(cmd redis.Cmder) bool {
	if cmd.Name() != "get" && cmd.Name() != "eval" {
		return false
	}

	for _, arg := range cmd.Args()[1:] {
		if arg == h.suite.key {
			return true
		}
	}

	return false
}

type leaderCounters struct {
	elected   atomic.Int32
	renounced atomic.Int32
}

type startNodeParams struct {
	ctx      context.Context
	counters *leaderCounters
	ttl      time.Duration
}

// startNode starts a single node and waits until it leads.
func (s *LeaderElectionSuite) startNode(p startNodeParams) *jobs.Node {
	counters := p.counters

	node := jobs.NewNode(jobs.Options{
		Key:        s.key,
		TTL:        p.ttl,
		Wait:       10 * time.Millisecond,
		OnLeader:   func(*jobs.Node) { counters.elected.Add(1) },
		OnRenounce: func(*jobs.Node) { counters.renounced.Add(1) },
	})

	node.Start(p.ctx)

	s.Require().Eventually(func() bool {
		return counters.elected.Load() == 1
	}, time.Second, 5*time.Millisecond)

	return node
}

func (s *LeaderElectionSuite) BeforeTest(_, _ string) {
	rediscache.Client().Del(context.Background(), s.key)
}

func (s *LeaderElectionSuite) Test_LeaderElection() {
	calledOnLeader := map[string]bool{}
	calledOnRenounce := map[string]bool{}
	nodes := []*jobs.Node{}

	ctx := context.Background()

	for i := 0; i <= 4; i++ {
		node := jobs.NewNode(jobs.Options{
			Key: s.key,
			OnLeader: func(n *jobs.Node) {
				calledOnLeader[n.ID()] = true
			},
			OnRenounce: func(n *jobs.Node) {
				calledOnRenounce[n.ID()] = true
			},
		})

		node.Start(ctx)
		nodes = append(nodes, node)
	}

	time.Sleep(time.Millisecond * 25)

	leaders := []*jobs.Node{}

	for i := 0; i <= 4; i++ {
		id := nodes[i].ID()

		if calledOnLeader[id] {
			leaders = append(leaders, nodes[i])
		}
	}

	s.Len(leaders, 1)

	// No one renounced their leadership yet
	for i := 0; i <= 4; i++ {
		s.False(calledOnRenounce[nodes[i].ID()])
	}

	// Reset
	calledOnLeader = map[string]bool{}
	calledOnRenounce = map[string]bool{}

	// Renounce the leadership
	leaders[0].Stop(ctx)

	s.True(calledOnRenounce[leaders[0].ID()])

	// There should be a new leader now
	time.Sleep(time.Second)
	s.Len(calledOnLeader, 1)
}

// Test_LostLeadership_Renounces verifies that a node whose key was taken over
// runs the renounce callback, so its master tasks are stopped.
func (s *LeaderElectionSuite) Test_LostLeadership_Renounces() {
	ctx := context.Background()
	counters := &leaderCounters{}
	node := s.startNode(startNodeParams{ctx: ctx, counters: counters, ttl: 200 * time.Millisecond})

	s.Require().NoError(rediscache.Client().Set(ctx, s.key, "another-node", time.Minute).Err())

	s.Eventually(func() bool {
		return counters.renounced.Load() == 1
	}, time.Second, 5*time.Millisecond)

	// The key stays with the other node, so this node must not lead again.
	time.Sleep(300 * time.Millisecond)

	s.Equal(int32(1), counters.elected.Load())
	s.Equal(int32(1), counters.renounced.Load())

	// The other node's lock must not have been touched by this node's renewal.
	s.Greater(rediscache.Client().PTTL(ctx, s.key).Val(), 30*time.Second)
	s.Equal("another-node", rediscache.Client().Get(ctx, s.key).Val())

	node.Stop(ctx)
}

// Test_RedisError_KeepsLeadership verifies that a single failed renewal is not
// treated as a lost leadership.
func (s *LeaderElectionSuite) Test_RedisError_KeepsLeadership() {
	ctx := context.Background()
	counters := &leaderCounters{}
	node := s.startNode(startNodeParams{ctx: ctx, counters: counters, ttl: time.Second})

	s.failRedis.Store(true)

	s.Require().Eventually(func() bool {
		return s.failed.Load() >= 1
	}, 2*time.Second, 5*time.Millisecond)

	s.failRedis.Store(false)
	time.Sleep(600 * time.Millisecond)

	s.Equal(int32(0), counters.renounced.Load())
	s.Equal(int32(1), counters.elected.Load())
	s.Equal(node.ID(), rediscache.Client().Get(ctx, s.key).Val())

	node.Stop(ctx)

	s.Equal(int32(1), counters.renounced.Load())
}

// Test_RedisError_RenouncesAfterTTL verifies that a node unable to renew for a
// full TTL renounces, since its key may have expired and been taken over.
func (s *LeaderElectionSuite) Test_RedisError_RenouncesAfterTTL() {
	ctx := context.Background()
	counters := &leaderCounters{}
	node := s.startNode(startNodeParams{ctx: ctx, counters: counters, ttl: 200 * time.Millisecond})

	s.failRedis.Store(true)

	s.Eventually(func() bool {
		return counters.renounced.Load() >= 1
	}, 2*time.Second, 5*time.Millisecond)

	s.GreaterOrEqual(s.failed.Load(), int32(2))

	s.failRedis.Store(false)
	node.Stop(ctx)
}

func TestLeaderElection(t *testing.T) {
	suite.Run(t, &LeaderElectionSuite{})
}
