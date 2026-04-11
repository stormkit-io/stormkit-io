package skauth

import (
	"context"
	"errors"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"golang.org/x/oauth2"
)

type emailClient struct{}

// NewEmailClient returns a Client implementation for the email/password provider.
// Email login does not use OAuth redirects; this client is used only to satisfy
// the Client interface and to signal that the provider is valid.
func NewEmailClient() Client {
	return &emailClient{}
}

func (c *emailClient) AuthCodeURL(_ AuthCodeURLParams) (string, error) {
	return "", errors.New("email provider does not support OAuth redirect flow")
}

func (c *emailClient) Exchange(_ context.Context, _ *shttp.RequestContext) (*oauth2.Token, error) {
	return nil, errors.New("email provider does not support OAuth token exchange")
}

func (c *emailClient) UserInfo(_ context.Context, _ *oauth2.Token) (*UserInfo, error) {
	return nil, errors.New("email provider does not support UserInfo via OAuth token")
}
