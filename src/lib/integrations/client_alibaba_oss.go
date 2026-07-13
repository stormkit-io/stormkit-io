//go:build alibaba

package integrations

import (
	"strings"
)

// GetFile uses AWS SDK under the hood to return the uploaded file.
func (a AlibabaClient) GetFile(args GetFileArgs) (*GetFileResult, error) {
	args.Location = strings.TrimPrefix(args.Location, "alibaba:")
	return a.awsClient.GetFile(args)
}
