package authwallhandlers

import "github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"

// StubMagicLinkEmail replaces sending sign-in links for the duration of a test.
// Sent links are passed to send synchronously. It returns a function that
// restores the real sender.
func StubMagicLinkEmail(available bool, send func(authwall.SendMagicLinkParams)) func() {
	origAvailable, origSend := magicLinkAvailable, sendMagicLink
	magicLinkAvailable = func() bool { return available }
	sendMagicLink = send

	return func() {
		magicLinkAvailable, sendMagicLink = origAvailable, origSend
	}
}
