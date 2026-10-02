package analytics

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

var sampleUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/30.0 Chrome/143.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
	"Googlebot/2.1 (+http://www.google.com/bot.html)",
	"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
	"curl/8.7.1",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/146.0.7680.165 Safari/537.36",
	"NetworkingExtension/8624.5.1.10.3 Network/5812.160.9 iOS/26.0",
	"GoogleAssociationService",
}

type BotVerdictCacheSuite struct {
	suite.Suite
}

// Test_CachedVerdictMatchesUncached verifies that memoisation never changes an
// answer: the second call for a user agent returns what classification did.
func (s *BotVerdictCacheSuite) Test_CachedVerdictMatchesUncached() {
	d := newBotDetector()

	for _, ua := range sampleUserAgents {
		expected := d.classify(ua)

		s.Equal(expected, d.isBot(ua), ua)
		s.Equal(expected, d.isBot(ua), "cached: %s", ua)
	}

	s.Equal(len(sampleUserAgents), d.verdicts.len())
}

func (s *BotVerdictCacheSuite) Test_EmptyUserAgentIsBotAndNotCached() {
	d := newBotDetector()

	s.True(d.isBot(""))
	s.Equal(0, d.verdicts.len())
}

// Test_EvictsLeastRecentlyUsed verifies the cache stays bounded and drops the
// entry that has gone longest without a lookup.
func (s *BotVerdictCacheSuite) Test_EvictsLeastRecentlyUsed() {
	c := newVerdictCache(3)

	c.put("a", true)
	c.put("b", false)
	c.put("c", true)

	// Touch "a" so "b" becomes the oldest.
	_, ok := c.get("a")
	s.True(ok)

	c.put("d", false)

	s.Equal(3, c.len())

	_, ok = c.get("b")
	s.False(ok, "b should have been evicted")

	for _, key := range []string{"a", "c", "d"} {
		_, ok = c.get(key)
		s.True(ok, "%s should still be cached", key)
	}
}

// Test_OversizedUserAgentIsNotCached verifies that a client cannot park large
// strings in the cache; it still gets a verdict.
func (s *BotVerdictCacheSuite) Test_OversizedUserAgentIsNotCached() {
	d := newBotDetector()
	ua := "Mozilla/5.0 " + strings.Repeat("x", maxCachedUserAgentLen)

	s.False(d.isBot(ua))
	s.Equal(0, d.verdicts.len())
}

func (s *BotVerdictCacheSuite) Test_ConcurrentAccess() {
	d := newBotDetector()

	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			for j := 0; j < 200; j++ {
				ua := sampleUserAgents[(i+j)%len(sampleUserAgents)]
				s.Equal(d.classify(ua), d.isBot(ua))

				d.isBot(fmt.Sprintf("scanner-%d-%d", i, j))
			}
		}(i)
	}

	wg.Wait()

	s.LessOrEqual(d.verdicts.len(), defaultVerdictCacheSize)
}

func TestBotVerdictCacheSuite(t *testing.T) {
	suite.Run(t, new(BotVerdictCacheSuite))
}

// BenchmarkIsBot_Uncached is the cost of one classification, which every
// request paid before the memo.
func BenchmarkIsBot_Uncached(b *testing.B) {
	d := newBotDetector()

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		d.classify(sampleUserAgents[i%len(sampleUserAgents)])
	}
}

// BenchmarkIsBot_Cached is the cost once the user agents have been seen.
func BenchmarkIsBot_Cached(b *testing.B) {
	d := newBotDetector()

	for _, ua := range sampleUserAgents {
		d.isBot(ua)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		d.isBot(sampleUserAgents[i%len(sampleUserAgents)])
	}
}
