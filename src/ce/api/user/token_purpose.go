package user

// Purpose identifies what a token was issued for. Every token carries one in
// its purpose claim, and is only accepted where that purpose is expected, so a
// token issued for one flow can never be used in another.
type Purpose string

// Tokens signed with the instance secret.
const (
	// PurposeSession is a dashboard login session.
	PurposeSession Purpose = "session"

	// PurposeOAuthState is the state of the dashboard's OAuth login flow.
	PurposeOAuthState Purpose = "oauth-state"

	// PurposeTeamInvite is a team invitation link.
	PurposeTeamInvite Purpose = "team-invite"

	// PurposeVolumeDownload is a volume file download link.
	PurposeVolumeDownload Purpose = "volume-download"

	// PurposeAuthWallForm is the token of an Auth Wall login form.
	PurposeAuthWallForm Purpose = "auth-wall-form"

	// PurposeAuthWallSession is an Auth Wall visitor session.
	PurposeAuthWallSession Purpose = "auth-wall-session"
)

// Tokens signed with an environment's Stormkit Auth secret.
const (
	// PurposeSkAuthSession is an end-user session.
	PurposeSkAuthSession Purpose = "skauth-session"

	// PurposeSkAuthAccessToken is an OAuth access token issued to a client.
	PurposeSkAuthAccessToken Purpose = "skauth-access-token"

	// PurposeSkAuthMagicLink is a magic link sign-in token.
	PurposeSkAuthMagicLink Purpose = "skauth-magic-link"

	// PurposeSkAuthEmailVerification is an email verification link.
	PurposeSkAuthEmailVerification Purpose = "skauth-email-verification"

	// PurposeSkAuthOAuthState is the state of an OAuth sign-in with a provider.
	PurposeSkAuthOAuthState Purpose = "skauth-oauth-state"
)

// purposeClaim is the claim that carries a token's purpose.
const purposeClaim = "purpose"
