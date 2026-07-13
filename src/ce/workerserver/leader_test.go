package jobs_test

import (
	"context"
	"testing"
	"time"

	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stretchr/testify/suite"
)

type LeaderElectionSuite struct {
	suite.Suite

	key string
}

func (s *LeaderElectionSuite) SetupSuite() {
	s.key = "leader_election_test"
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

func TestLeaderElection(t *testing.T) {
	suite.Run(t, &LeaderElectionSuite{})
}
