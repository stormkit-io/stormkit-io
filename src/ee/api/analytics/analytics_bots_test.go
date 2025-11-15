package analytics_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stretchr/testify/suite"
)

type BotsSuite struct {
	suite.Suite
}

func (s *BotsSuite) Test_IsBot_DetectsKnownBots() {
	botUserAgents := []string{
		"Googlebot/2.1 (+http://www.google.com/bot.html)",
		"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
		"facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)",
		"Twitterbot/1.0",
		"LinkedInBot/1.0 (compatible; Mozilla/5.0; Apache-HttpClient +http://www.linkedin.com/)",
		"Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)",
		"WhatsApp/2.19.81 A",
		"TelegramBot (like TwitterBot)",
		"DiscordBot (https://discordapp.com)",
		"crawler",
		"spider",
		"bot",
		"scraper",
	}

	for _, userAgent := range botUserAgents {
		s.True(analytics.IsBot(userAgent), "Should detect '%s' as a bot", userAgent)
	}
}

func (s *BotsSuite) Test_IsBot_AllowsRealUsers() {
	realUserAgents := []string{
		"Mozilla/5.0 (Linux; Android 6.0; LG-H810 Build/MRA58K; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/88.0.4324.93 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 10; Infinix X656) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.66 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 9; Redmi 8 Build/PKQ1.190319.001) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.106 Mobile Safari/537.36 YaApp_Android/10.91 YaSearchBrowser/10.91",
		"Mozilla/5.0 (Linux; Android 8.1.0; SM-J260Y Build/M1AJB; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/89.0.4389.86 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 9; CPH1937) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.127 Mobile Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36",
		"Mozilla/5.0 (iPad; CPU OS 14_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 [FBAN/FBIOS;FBDV/iPad7,5;FBMD/iPad;FBSN/iOS;FBSV/14.2;FBSS/2;FBID/tablet;FBLC/de_DE;FBOP/5]",
		"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.45 Safari/537.36",
		"Mozilla/5.0 (Linux; Android 6.0.1; XT1254) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.96 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 9; vivo 1901) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.83 Mobile Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36/LN5CEYvR-53",
		"Mozilla/5.0 (Linux; Android 7.0; TECNO F3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Mobile Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.2.3988.123 Safari/537.36",
		"Dalvik/1.6.0 (Linux; U; Android 6.0; SM-G360H Build/KTU84P)",
		"Dalvik/2.1.0 (Linux; U; Android 10; moto g play (2021) Build/QZAS30.Q4-39-39-1-13)",
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:77.0) Gecko/20100101 Firefox/77.0,gzip(gfe)",
		"Mozilla/5.0 (Linux; Android 11; KB2003) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.50 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 11; SM-M115F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.159 Mobile Safari/537.36",
		"Dalvik/2.1.0 (Linux; U; Android 5.1; S4T Build/LMY47D)",
		"Dalvik/1.6.0 (Linux; U; Android 4.4.4; SMART Build/89fcb58e_20190304_114404)",
		"Mozilla/5.0 (Windows NT 6.2; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.55 Safari/537.36 Edg/96.0.1054.34",
		"Mozilla/5.0 (Linux; Android 8.1.0; DUA-L22) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.141 Mobile Safari/537.36",
		"Mozilla/5.0 (iPad; CPU OS 8_1 like Mac OS X) AppleWebKit/600.1.4 (KHTML, like Gecko) Version/8.0 YaBrowser/16.9.0.3310.11 Mobile/12B410 Safari/600.1.4",
		"Dalvik/2.1.0 (Linux; U; Android 5.1.1; Fire Build/LMY49M)",
		"Mozilla/5.0 (Linux; Android 6.0; STARTRAIL 8 Build/MRA58K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/52.0.2743.98 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; arm_64; Android 10; SM-G975F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.85 YaApp_Android/21.113.1 YaSearchBrowser/21.113.1 BroPP/1.0 SA/3 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 7.0; LG-H831) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.90 Mobile Safari/537.36",
		"Dalvik/2.1.0 (Linux; U; Android 8.1.0; itel A15 Build/O11019)",
		"Mozilla/5.0 (Linux; Android 9; CPH1951 Build/PPR1.180610.011; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/83.0.4103.101 Mobile Safari/537.36 Viber/13.2.0.8",
		"Mozilla/5.0 (Linux; Android 10; 5061K_EEA) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.71 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 5.1.1; Woxter Nimbus1100RX Build/LMY48B; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/83.0.4103.106 Safari/537.36",
		"Mozilla/5.0 (Linux; Android 7.0; CASPER_VIA_M3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.83 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 7.0; XT1585 Build/NCK25.118-10.5; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/78.0.3904.90 Mobile Safari/537.36",
		"Dalvik/2.1.0 (Linux; U; Android 9; LAVA LH9930 Build/PPR1.180610.011)",
		"Mozilla/5.0 (Linux; Android 8.0.0; SM-G950F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.90 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 8.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/88.0.4324.93 Mobile DuckDuckGo/5 Safari/537.36",
		"Mozilla/5.0 (Linux; Android 8.1.0; vivo 1807 Build/OPM1.171019.026; wv) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.84 Mobile Safari/537.36 VivoBrowser/6.3.0.5",
		"Dalvik/2.1.0 (Linux; U; Android 7.0; X20S Build/NRD90M)",
		"Mozilla/5.0 (Linux; Android 11; GM1917) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 11; M2101K7BG) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.61 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 10; TECNO KE6j) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.101 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 7.0; PMT3537_4G Build/NRD90M) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.111 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 8.1.0) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/93.0.4577.62 DuckDuckGo/5 Safari/537.36",
		"Mozilla/5.0 (Linux; Android 9; MI 6 Build/PKQ1.190118.001) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.106 Mobile Safari/537.36 YaApp_Android/10.91 YaSearchBrowser/10.91",
		"Mozilla/5.0 (Linux; x86; Android 5.1; Hi10 plus) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.96 YaBrowser/20.4.1.144.01 Safari/537.36",
		"Mozilla/5.0 (Linux; Android 10; SM-G960W) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.101 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 9; CPH1931 Build/PKQ1.190714.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/89.0.4389.105 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 10; ANA-AN00; HMSCore 4.0.4.307) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 HuaweiBrowser/10.1.2.300 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 11; SM-S127DL) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.4577.62 Mobile Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (iPad; CPU OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Android 11; Mobile; rv:89.0) Gecko/89.0 Firefox/89.0",
		"PostmanRuntime/7.28.0",
	}

	for _, userAgent := range realUserAgents {
		s.False(analytics.IsBot(userAgent), "Should not detect '%s' as a bot", userAgent)
	}
}

func (s *BotsSuite) Test_IsBot_CaseInsensitive() {
	testCases := []string{
		"GoogleBot/2.1",
		"GOOGLEBOT/2.1",
		"googlebot/2.1",
		"Bot test",
		"BOT TEST",
		"bot test",
		"MJ12bot",
		"Mail.RU_Bot",
		"NaverBot",
		"Nimbostratus-Bot",
		"NinjaBot",
		"rogerBot",
		"Roverbot",
		"YisouSpider",
		"ZoomSpider",
		"CrazyWebCrawler",
		"gsa-crawler",
		"ICC-Crawler",
		"magpie-crawler",
		"Sogou web spider",
		"Spider",
		"EtaoSpider",
		"Honeybadger Uptime Check",
	}

	for _, userAgent := range testCases {
		s.True(analytics.IsBot(userAgent), "Should detect '%s' as a bot (case insensitive)", userAgent)
	}
}

func TestBotsSuite(t *testing.T) {
	suite.Run(t, &BotsSuite{})
}
