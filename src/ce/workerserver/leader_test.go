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

	key      string
	failGets atomic.Bool
	failed   atomic.Int32
}

func (s *LeaderElectionSuite) SetupSuite() {
	s.key = "leader_election_test"
	rediscache.Client().AddHook(&failingGetHook{suite: s})
}

func (s *LeaderElectionSuite) TearDownTest() {
	s.failGets.Store(false)
	s.failed.Store(0)
}

// failingGetHook fails GET commands on the suite's key while failGets is set,
// simulating a Redis blip for the leader check only.
type failingGetHook struct {
	suite *LeaderElectionSuite
}

func (h *failingGetHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *failingGetHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		args := cmd.Args()

		if h.suite.failGets.Load() && cmd.Name() == "get" && len(args) > 1 && args[1] == h.suite.key {
			err := errors.New("simulated redis failure")
			cmd.SetErr(err)
			h.suite.failed.Add(1)

			return err
		}

		return next(ctx, cmd)
	}
}

func (h *failingGetHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

type leaderCounters struct {
	elected   atomic.Int32
	renounced atomic.Int32
}

// startNode starts a single node with a short TTL and waits until it leads.
func (s *LeaderElectionSuite) startNode(ctx context.Context, counters *leaderCounters) *jobs.Node {
	node := jobs.NewNode(jobs.Options{
		Key:        s.key,
		TTL:        200 * time.Millisecond,
		Wait:       10 * time.Millisecond,
		OnLeader:   func(*jobs.Node) { counters.elected.Add(1) },
		OnRenounce: func(*jobs.Node) { counters.renounced.Add(1) },
	})

	node.Start(ctx)

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
	node := s.startNode(ctx, counters)

	s.Require().NoError(rediscache.Client().Set(ctx, s.key, "another-node", time.Minute).Err())

	s.Eventually(func() bool {
		return counters.renounced.Load() == 1
	}, time.Second, 5*time.Millisecond)

	// The key stays with the other node, so this node must not lead again.
	time.Sleep(300 * time.Millisecond)

	s.Equal(int32(1), counters.elected.Load())
	s.Equal(int32(1), counters.renounced.Load())

	node.Stop(ctx)
}

// Test_GetError_KeepsLeadership verifies that a failed leader check is not
// treated as a lost leadership.
func (s *LeaderElectionSuite) Test_GetError_KeepsLeadership() {
	ctx := context.Background()
	counters := &leaderCounters{}
	node := s.startNode(ctx, counters)

	// Keep the key alive while renewals are skipped, so only the read fails.
	s.Require().NoError(rediscache.Client().PExpire(ctx, s.key, time.Minute).Err())
	s.failGets.Store(true)

	s.Require().Eventually(func() bool {
		return s.failed.Load() >= 2
	}, 2*time.Second, 5*time.Millisecond)

	s.failGets.Store(false)
	time.Sleep(250 * time.Millisecond)

	s.Equal(int32(0), counters.renounced.Load())
	s.Equal(int32(1), counters.elected.Load())
	s.Equal(node.ID(), rediscache.Client().Get(ctx, s.key).Val())

	node.Stop(ctx)

	s.Equal(int32(1), counters.renounced.Load())
}

func TestLeaderElection(t *testing.T) {
	suite.Run(t, &LeaderElectionSuite{})
}
