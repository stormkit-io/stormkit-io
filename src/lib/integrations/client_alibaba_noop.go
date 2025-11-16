//go:build !alibaba

package integrations

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

func init() {
	slog.Debug(slog.LogOpts{
		Msg:   "alibaba integration is disabled (noop client)",
		Level: slog.DL1,
	})
}

type AlibabaSDK interface {
	Upload()
}
type AlibabaClient struct{}

var CachedAlibabaClient *AlibabaClient
var DefaultAlibabaSDK any

func Alibaba(args ClientArgs) (*AlibabaClient, error) {
	return &AlibabaClient{}, nil
}

func (a *AlibabaClient) Upload(args UploadArgs) (*UploadResult, error) {
	return nil, nil
}

func (a *AlibabaClient) GetFile(args GetFileArgs) (*GetFileResult, error) {
	return nil, nil
}

func (a *AlibabaClient) Invoke(args InvokeArgs) (*InvokeResult, error) {
	return nil, nil
}

func (a *AlibabaClient) Name() string {
	return "alibaba-noop"
}

func (a *AlibabaClient) DeleteArtifacts(ctx context.Context, args DeleteArtifactsArgs) error {
	return nil
}
