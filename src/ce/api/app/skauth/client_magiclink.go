package skauth

import (
	"context"
	"errors"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"golang.org/x/oauth2"
)

type magicLinkClient struct{}

// NewMagicLinkClient returns a Client implementation for the magic link provider.
// Magic link login does not use OAuth redirects; this client is used only to satisfy
// the Client interface and to signal that the provider is valid.
func NewMagicLinkClient() Client {
	return &magicLinkClient{}
}

func (c *magicLinkClient) AuthCodeURL(_ AuthCodeURLParams) (string, error) {
	return "", errors.New("magiclink provider does not support OAuth redirect flow")
}

func (c *magicLinkClient) Exchange(_ context.Context, _ *shttp.RequestContext) (*oauth2.Token, error) {
	return nil, errors.New("magiclink provider does not support OAuth token exchange")
}

func (c *magicLinkClient) UserInfo(_ context.Context, _ *oauth2.Token) (*UserInfo, error) {
	return nil, errors.New("magiclink provider does not support UserInfo via OAuth token")
}
