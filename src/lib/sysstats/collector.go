package sysstats

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultExporterPort is node_exporter's listening port.
const DefaultExporterPort = "9100"

// defaultTimeout bounds a single scrape. node_exporter answers in milliseconds
// on a healthy host, so anything slower is a machine worth flagging rather than
// waiting on.
const defaultTimeout = 5 * time.Second

// Collector scrapes node_exporter endpoints. It keeps the previous CPU counters
// per target, so it must be long-lived rather than constructed per scrape.
type Collector struct {
	client *http.Client
	mu     sync.Mutex
	prev   map[string]cpuCounters
}

type NewCollectorParams struct {
	Timeout time.Duration
	Client  *http.Client
}

func NewCollector(p NewCollectorParams) *Collector {
	client := p.Client

	if client == nil {
		timeout := p.Timeout

		if timeout == 0 {
			timeout = defaultTimeout
		}

		client = &http.Client{Timeout: timeout}
	}

	return &Collector{
		client: client,
		prev:   map[string]cpuCounters{},
	}
}

// TargetURL builds the metrics URL for a target. A target may be a bare host, a
// host:port, or a full URL.
func TargetURL(target string) string {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return strings.TrimSuffix(target, "/") + "/metrics"
	}

	if !strings.Contains(target, ":") {
		target = fmt.Sprintf("%s:%s", target, DefaultExporterPort)
	}

	return fmt.Sprintf("http://%s/metrics", target)
}

// Collect scrapes a single target. An unreachable target yields a Sample with
// Reachable false and the reason set, never a nil Sample — a machine that
// stopped answering is a fact worth recording and showing.
func (c *Collector) Collect(ctx context.Context, target string) *Sample {
	now := time.Now().Unix()

	sample, err := c.scrape(ctx, target)

	if err != nil {
		return &Sample{
			Timestamp: now,
			Target:    target,
			Reachable: false,
			Error:     err.Error(),
		}
	}

	sample.Timestamp = now
	sample.Target = target

	return sample
}

func (c *Collector) scrape(ctx context.Context, target string) (*Sample, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, TargetURL(target), nil)

	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exporter returned status %d", res.StatusCode)
	}

	p, err := newParser(res.Body)

	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var prev *cpuCounters

	if p, ok := c.prev[target]; ok {
		prev = &p
	}

	sample, counters := p.sample(prev)
	c.prev[target] = counters

	return sample, nil
}

// Forget drops the retained CPU counters for a target. Callers use it when a
// target disappears so a machine returning later starts from a clean delta
// instead of one spanning its absence.
func (c *Collector) Forget(target string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.prev, target)
}
