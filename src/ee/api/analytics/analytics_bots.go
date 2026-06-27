package analytics

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

// crawlerUserAgents is the vendored monperrus/crawler-user-agents dataset (MIT,
// see crawler_user_agents_LICENSE). Refresh it with:
//
//	go generate ./src/ee/api/analytics/...
//
//go:generate sh -c "curl -sSL https://raw.githubusercontent.com/monperrus/crawler-user-agents/master/crawler-user-agents.json -o crawler_user_agents.json"
//go:embed crawler_user_agents.json
var crawlerUserAgents []byte

// knownBots is a supplementary, hand-curated list of agents found in our own
// analytics database that the vendored crawler dataset does not cover — mostly
// uptime monitors and niche preview/link checkers. General crawlers and generic
// HTTP clients are intentionally NOT listed here; they are handled by
// crawlerUserAgents and keywordPatterns respectively. Matched as
// case-insensitive substrings.
var knownBots = []string{
	"amazon-kendra",
	"ask jeeves",
	"curious george",
	"FeedDemon",
	"GitLab",
	"GoodLinks",
	"infoseek",
	"Lenns.io",
	"LinkValidator",
	"lychee", // https://github.com/lycheeverse/lychee
	"Lycos",
	"ManicTime",
	"Mozlila", // https://trunc.org/learning/the-mozlila-user-agent-bot
	"msray-plus",
	"NetworkingExtension",
	"Pulsetic.com",
	"search.marginalia.nu",
	"Stormkit", // our own domain health pinger (StormkitBot) and internal agents
	"Teleport Pro",
	"TeleportPro",
	"Tines",
	"upptime.js.org",
	"XML-Sitemaps",
	"Y!J-ASR",
	"Y!J-BSC",
}

// overBroadCrawlerPatterns are vendored patterns we drop because their bare
// token also appears in real in-app browser user agents (e.g. the Viber
// webview), which we count as genuine visitors rather than bots.
var overBroadCrawlerPatterns = map[string]bool{
	"Viber": true,
}

// keywordPatterns catch generic non-browser clients and automation tooling that
// are not enumerated by name in either list above.
var keywordPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(bot|crawler|spider|scraper|fetch|monitor|check|test)\b`),
	regexp.MustCompile(`(?i)\b(curl|wget|http|client|java|python|go-http|ruby|php)\b`),
	regexp.MustCompile(`(?i)\b(headless|phantom|selenium|playwright)\b`),
	regexp.MustCompile(`(?i)\b(uptime|monitor|ping|health|status)\b`),
}

// botDetector classifies user agents as bots using three independent layers:
// the vendored crawler dataset, a hand-curated supplementary list, and generic
// keyword heuristics. Any layer matching is sufficient.
type botDetector struct {
	crawlerPattern  *regexp.Regexp
	keywordPatterns []*regexp.Regexp
	knownBots       []string
}

func newBotDetector() *botDetector {
	d := &botDetector{
		keywordPatterns: keywordPatterns,
		knownBots:       make([]string, len(knownBots)),
	}

	for i, bot := range knownBots {
		d.knownBots[i] = strings.ToLower(bot)
	}

	d.crawlerPattern = compileCrawlerPattern(crawlerUserAgents)

	return d
}

// compileCrawlerPattern parses the vendored dataset and combines every valid
// pattern into a single case-insensitive alternation, so matching is one regex
// execution per request rather than one per entry. Patterns that fail to
// compile (e.g. non-RE2 constructs) are skipped individually.
func compileCrawlerPattern(data []byte) *regexp.Regexp {
	var entries []struct {
		Pattern string `json:"pattern"`
	}

	if err := json.Unmarshal(data, &entries); err != nil {
		slog.Errorf("could not parse crawler user agents dataset: %v", err)
		return nil
	}

	valid := make([]string, 0, len(entries))
	seen := map[string]bool{}

	for _, entry := range entries {
		if entry.Pattern == "" || seen[entry.Pattern] || overBroadCrawlerPatterns[entry.Pattern] {
			continue
		}

		if _, err := regexp.Compile(entry.Pattern); err != nil {
			continue
		}

		seen[entry.Pattern] = true
		valid = append(valid, "(?:"+entry.Pattern+")")
	}

	if len(valid) == 0 {
		return nil
	}

	combined, err := regexp.Compile("(?i)" + strings.Join(valid, "|"))

	if err != nil {
		slog.Errorf("could not compile combined crawler pattern: %v", err)
		return nil
	}

	return combined
}

func (d *botDetector) isBot(userAgent string) bool {
	if userAgent == "" {
		return true
	}

	if d.crawlerPattern != nil && d.crawlerPattern.MatchString(userAgent) {
		return true
	}

	for _, pattern := range d.keywordPatterns {
		if pattern.MatchString(userAgent) {
			return true
		}
	}

	lower := strings.ToLower(userAgent)

	for _, bot := range d.knownBots {
		if strings.Contains(lower, bot) {
			return true
		}
	}

	return false
}

var defaultBotDetector = newBotDetector()

// IsBot reports whether the user agent belongs to a known crawler, automation
// tool, or other non-human client that should be excluded from analytics.
func IsBot(userAgent string) bool {
	return defaultBotDetector.isBot(userAgent)
}

func IsUtf8(str string) bool {
	data := []byte(str)

	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)

		if r == utf8.RuneError && size == 1 {
			return false
		}

		data = data[size:]
	}

	return true
}
