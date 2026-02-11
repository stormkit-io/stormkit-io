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
	redis      *rediscache.RedisCache
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
		redis:      rediscache.Client(),
		options:    options,
		key:        options.Key,
		stopChan:   make(chan struct{}),
		onLeader:   options.OnLeader,
		onRenounce: options.OnRenounce,
		onStart:    options.OnStart,
	}
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

	l.isLeader(ctx, func(isLeader bool) {
		if isLeader {
			_, err := l.redis.Do(ctx, "DEL", l.key).Result()

			if err != nil {
				slog.Errorf("error while revoking key: %s", err.Error())
			}

			if l.onRenounce != nil {
				l.onRenounce(l)
			}
		}
	})
}

func (l *Node) elect(ctx context.Context) {
	for {
		select {
		case <-l.stopChan:
			return
		default:
			reply, err := l.redis.Do(ctx, "SET", l.key, l.id, "PX", int(l.options.TTL/time.Millisecond), "NX").Result()

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
				l.renewMu.Lock()
				l.renewID = time.NewTicker(l.options.TTL / 2)
				l.renewMu.Unlock()

				go l.renew(ctx)

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

func (l *Node) renew(ctx context.Context) {
	for {
		select {
		case <-l.stopChan:
			return
		case <-l.renewID.C:
			l.isLeader(ctx, func(isLeader bool) {
				if isLeader {
					_, err := l.redis.Do(ctx, "PEXPIRE", l.key, int(l.options.TTL/time.Millisecond)).Result()
					if err != nil {
						slog.Errorf("failed to renew leader key: %v", err)
					}
				} else {
					l.renewMu.Lock()
					l.renewID.Stop()
					l.renewMu.Unlock()

					time.AfterFunc(l.options.Wait, func() { l.elect(ctx) })
					return
				}
			})
		}
	}
}

func (l *Node) isLeader(ctx context.Context, callback func(bool)) {
	reply, err := l.redis.Do(ctx, "GET", l.key).Result()

	if err != nil && err != redis.Nil {
		slog.Errorf("failed to get leader key: %v", err)
		callback(false)
		return
	}

	callback(reply == l.id)
}
