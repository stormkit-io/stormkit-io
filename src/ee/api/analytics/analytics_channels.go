package analytics

const (
	ChannelDirect   = "direct"
	ChannelInternal = "internal"
	ChannelAI       = "ai"
	ChannelEmail    = "email"
	ChannelSearch   = "search"
	ChannelSocial   = "social"
	ChannelReferral = "referral"
)

// channelPatterns classify a referrer host. The rollup job strips the scheme,
// path and a leading "www." first, so patterns are written without them. The lists are checked in order, so a more specific host
// (gemini.google.com, mail.google.com) is matched before a general one. A host
// matching none is a referral.
var channelPatterns = []struct {
	channel  string
	patterns []string
}{
	{ChannelAI, []string{
		`(^|\.)chatgpt\.com$`,
		`^chat\.openai\.com$`,
		`(^|\.)perplexity\.ai$`,
		`^claude\.ai$`,
		`^gemini\.google\.com$`,
		`^copilot\.microsoft\.com$`,
		`^chat\.deepseek\.com$`,
		`^chat\.mistral\.ai$`,
		`^grok\.com$`,
		`^meta\.ai$`,
		`(^|\.)you\.com$`,
		`(^|\.)phind\.com$`,
		`^poe\.com$`,
	}},
	{ChannelEmail, []string{
		`^mail\.google\.com$`,
		`^outlook\.(live|office|office365)\.com$`,
		`^mail\.yahoo\.com$`,
		`^mail\.proton\.me$`,
		`^com\.google\.android\.gm$`,
	}},
	{ChannelSearch, []string{
		// Only the search front page: accounts, docs, sites and news are other
		// Google products, and a sign-in redirect is not a search visit.
		`^google\.[a-z]{2,3}(\.[a-z]{2})?$`,
		`^com\.google\.android\.googlequicksearchbox$`,
		`(^|\.)bing\.com$`,
		`(^|\.)duckduckgo\.com$`,
		`(^|\.)search\.yahoo\.com$`,
		`(^|\.)yandex\.[a-z]{2,3}$`,
		`(^|\.)baidu\.com$`,
		`(^|\.)ecosia\.org$`,
		`^search\.brave\.com$`,
		`(^|\.)qwant\.com$`,
		`(^|\.)startpage\.com$`,
		`(^|\.)kagi\.com$`,
		`(^|\.)naver\.com$`,
		`(^|\.)seznam\.cz$`,
	}},
	{ChannelSocial, []string{
		`^t\.co$`,
		`(^|\.)x\.com$`,
		`(^|\.)twitter\.com$`,
		`(^|\.)facebook\.com$`,
		`^fb\.me$`,
		`(^|\.)instagram\.com$`,
		`(^|\.)linkedin\.com$`,
		`^lnkd\.in$`,
		`(^|\.)reddit\.com$`,
		`^news\.ycombinator\.com$`,
		`(^|\.)youtube\.com$`,
		`^youtu\.be$`,
		`(^|\.)tiktok\.com$`,
		`(^|\.)pinterest\.[a-z.]+$`,
		`(^|\.)threads\.net$`,
		`^bsky\.app$`,
		`(^|\.)mastodon\.social$`,
		`(^|\.)discord\.com$`,
		`^t\.me$`,
		`(^|\.)producthunt\.com$`,
		`(^|\.)indiehackers\.com$`,
		`^dev\.to$`,
	}},
}

// ChannelPatterns returns the referrer host patterns of a channel, for the
// rollup job that classifies referrers.
func ChannelPatterns(channel string) []string {
	for _, c := range channelPatterns {
		if c.channel == channel {
			return c.patterns
		}
	}

	return nil
}
