package analytics

import (
	"container/list"
	"sync"
)

const (
	// defaultVerdictCacheSize bounds the memo. A busy edge sees a few hundred
	// distinct user agents an hour, so the room is mostly slack for scanners
	// that rotate agents. One that burns through all of it between two visits
	// from the same real agent only costs that agent one re-classification.
	defaultVerdictCacheSize = 4096

	// maxCachedUserAgentLen keeps a client from parking large strings in the
	// cache. Anything longer is classified every time, which is what happened
	// to every request before the cache existed.
	maxCachedUserAgentLen = 1024
)

// verdictCache remembers whether a user agent was classified as a bot. Least
// recently used entries are evicted once the cache is full.
type verdictCache struct {
	mu      sync.Mutex
	size    int
	entries map[string]*list.Element
	order   *list.List // front is most recently used
}

type verdictEntry struct {
	userAgent string
	isBot     bool
}

func newVerdictCache(size int) *verdictCache {
	return &verdictCache{
		size:    size,
		entries: make(map[string]*list.Element, size),
		order:   list.New(),
	}
}

func (c *verdictCache) get(userAgent string) (isBot, ok bool) {
	if c == nil || c.size <= 0 {
		return false, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	element, found := c.entries[userAgent]

	if !found {
		return false, false
	}

	c.order.MoveToFront(element)

	return element.Value.(*verdictEntry).isBot, true
}

func (c *verdictCache) put(userAgent string, isBot bool) {
	if c == nil || c.size <= 0 || len(userAgent) > maxCachedUserAgentLen {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if element, found := c.entries[userAgent]; found {
		element.Value.(*verdictEntry).isBot = isBot
		c.order.MoveToFront(element)

		return
	}

	for c.order.Len() >= c.size {
		oldest := c.order.Back()
		c.order.Remove(oldest)
		delete(c.entries, oldest.Value.(*verdictEntry).userAgent)
	}

	c.entries[userAgent] = c.order.PushFront(&verdictEntry{userAgent: userAgent, isBot: isBot})
}

func (c *verdictCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.order.Len()
}
