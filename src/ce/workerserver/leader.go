package jobs

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

// Node represents the leader election structure
type Node struct {
	id         string
	options    Options
	key        string
	renewMu    sync.Mutex
	renewID    *time.Ticker
	electID    *time.Timer
	onLeader   func(node *Node)
	onRenounce func(node *Node)
	onStart    func(node *Node)
	stopChan   chan struct{}
}

// Options represents the configuration options for the leader election
type Options struct {
	TTL        time.Duration
	Wait       time.Duration
	Key        string
	OnStart    func(node *Node)
	OnLeader   func(node *Node)
	OnRenounce func(node *Node)
}

// NewNode initializes a new Leader instance
func NewNode(options Options) *Node {
	if options.TTL == 0 {
		options.TTL = 10 * time.Second
	}

	if options.Wait == 0 {
		options.Wait = 1 * time.Second
	}

	if options.Key == "" {
		options.Key = "leader-election"
	}

	return &Node{
		id:         uuid.New().String(),
		options:    options,
		key:        options.Key,
		stopChan:   make(chan struct{}),
		onLeader:   options.OnLeader,
		onRenounce: options.OnRenounce,
		onStart:    options.OnStart,
	}
}

// do runs a Redis command against the shared client.
//
// The client is resolved per call rather than held on the node, so a client
// discarded after a failover is replaced instead of being reused once closed.
func (l *Node) do(ctx context.Context, args ...any) (any, error) {
	client := rediscache.Client()

	if client == nil {
		return nil, errors.New("redis client is not available")
	}

	// Recovery from a failover is handled by the client itself, so this only
	// has to avoid holding a client that recovery has replaced.
	return client.Do(ctx, args...).Result()
}

// ID returns the node id.
func (l *Node) ID() string {
	return l.id
}

// OnStart registers a callback function that will be executed
// when the node starts.
func (l *Node) OnStart(cb func(n *Node)) *Node {
	l.onStart = cb
	return l
}

// OnLeaderRenounced registers a callback function that will
// be executed when the leadership is renounced.
func (l *Node) OnLeaderRenounced(cb func(n *Node)) *Node {
	l.onRenounce = cb
	return l
}

// OnLeaderElected sets the leader election callback function.
// This should be called before the `Start` method.
func (l *Node) OnLeaderElected(cb func(n *Node)) *Node {
	l.onLeader = cb
	return l
}

// Start begins the leader election process
func (l *Node) Start(ctx context.Context) {
	go l.elect(ctx)

	if l.onStart != nil {
		l.onStart(l)
	}
}

// Stop stops the leader election process
func (l *Node) Stop(ctx context.Context) {
	l.stopChan <- struct{}{}
	l.renewMu.Lock()
	defer l.renewMu.Unlock()

	if l.renewID != nil {
		l.renewID.Stop()
	}

	if l.electID != nil {
		l.electID.Stop()
	}

	if l.leadership(ctx) == leadershipHeld {
		_, err := l.do(ctx, "DEL", l.key)

		if err != nil {
			slog.Errorf("error while revoking key: %s", err.Error())
		}

		if l.onRenounce != nil {
			l.onRenounce(l)
		}
	}
}

func (l *Node) elect(ctx context.Context) {
	for {
		select {
		case <-l.stopChan:
			return
		default:
			reply, err := l.do(ctx, "SET", l.key, l.id, "PX", int(l.options.TTL/time.Millisecond), "NX")

			if rediscache.IsConnectionError(err) {
				slog.Errorf("redis connection error: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			if err != nil && !errors.Is(err, redis.Nil) {
				slog.Errorf("failed to set leader key: %v", err)
				time.Sleep(l.options.Wait)
				continue
			}

			if reply == "OK" {
				ticker := time.NewTicker(l.options.TTL / 2)

				l.renewMu.Lock()
				l.renewID = ticker
				l.renewMu.Unlock()

				go l.renew(ctx, ticker)

				if l.onLeader != nil {
					l.onLeader(l)
				}

				return
			} else {
				time.Sleep(l.options.Wait)
			}
		}
	}
}

// renew keeps the leader key alive on every tick. It returns once the
// leadership is lost, after renouncing it and scheduling a new election.
func (l *Node) renew(ctx context.Context, ticker *time.Ticker) {
	for {
		select {
		case <-l.stopChan:
			return
		case <-ticker.C:
			switch l.leadership(ctx) {
			case leadershipHeld:
				_, err := l.do(ctx, "PEXPIRE", l.key, int(l.options.TTL/time.Millisecond))

				if err != nil {
					slog.Errorf("failed to renew leader key: %v", err)
				}

			case leadershipUnknown:
				// A failed read says nothing about who holds the key, so keep
				// leading and check again on the next tick. If the outage outlasts
				// the TTL, the key expires and the next successful read reports
				// the loss.

			case leadershipLost:
				l.renewMu.Lock()
				ticker.Stop()
				l.renewMu.Unlock()

				if l.onRenounce != nil {
					l.onRenounce(l)
				}

				time.AfterFunc(l.options.Wait, func() { l.elect(ctx) })
				return
			}
		}
	}
}

type leadershipState int

const (
	leadershipHeld leadershipState = iota
	leadershipLost
	leadershipUnknown
)

// leadership reports whether this node holds the leader key. A Redis error
// other than a missing key yields leadershipUnknown.
func (l *Node) leadership(ctx context.Context) leadershipState {
	reply, err := l.do(ctx, "GET", l.key)

	if errors.Is(err, redis.Nil) {
		return leadershipLost
	}

	if err != nil {
		slog.Errorf("failed to get leader key: %v", err)
		return leadershipUnknown
	}

	if reply == l.id {
		return leadershipHeld
	}

	return leadershipLost
}
