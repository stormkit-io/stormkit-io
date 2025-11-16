//go:build !alibaba

package integrations_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type AlibabaNoopSuite struct {
	suite.Suite
	client *integrations.AlibabaClient
}

func (s *AlibabaNoopSuite) SetupSuite() {
	integrations.CachedAlibabaClient = nil
}

func (s *AlibabaNoopSuite) BeforeTest(_, _ string) {
	integrations.CachedAlibabaClient = nil
	var err error
	s.client, err = integrations.Alibaba(integrations.ClientArgs{})
	s.NoError(err)
	s.NotNil(s.client)
}

func (s *AlibabaNoopSuite) AfterTest(_, _ string) {
	integrations.CachedAlibabaClient = nil
}

func (s *AlibabaNoopSuite) Test_Client_Name() {
	name := s.client.Name()
	s.Equal("alibaba-noop", name)
}

func (s *AlibabaNoopSuite) Test_Upload_ReturnsNil() {
	result, err := s.client.Upload(integrations.UploadArgs{
		AppID:        123,
		DeploymentID: types.ID(456),
		ClientZip:    "/path/to/client.zip",
		ServerZip:    "/path/to/server.zip",
		APIZip:       "/path/to/api.zip",
	})

	s.NoError(err)
	s.Nil(result, "noop client should return nil for Upload")
}

func (s *AlibabaNoopSuite) Test_GetFile_ReturnsNil() {
	result, err := s.client.GetFile(integrations.GetFileArgs{
		Location:     "alibaba:bucket/app/deployment",
		FileName:     "test.html",
		DeploymentID: types.ID(456),
	})

	s.NoError(err)
	s.Nil(result, "noop client should return nil for GetFile")
}

func (s *AlibabaNoopSuite) Test_Invoke_ReturnsNil() {
	result, err := s.client.Invoke(integrations.InvokeArgs{
		ARN:    "arn:test:function",
		Method: "GET",
	})

	s.NoError(err)
	s.Nil(result, "noop client should return nil for Invoke")
}

func (s *AlibabaNoopSuite) Test_DeleteArtifacts_NoError() {
	ctx := context.Background()

	err := s.client.DeleteArtifacts(ctx, integrations.DeleteArtifactsArgs{
		StorageLocation:  "alibaba:bucket/app/deployment",
		FunctionLocation: "arn:function:123",
		APILocation:      "arn:api:456",
	})

	s.NoError(err, "noop client should not return error for DeleteArtifacts")
}

func (s *AlibabaNoopSuite) Test_CachedClient_ReturnsAlready() {
	// First call
	client1, err := integrations.Alibaba(integrations.ClientArgs{})
	s.NoError(err)
	s.NotNil(client1)

	// Second call should return cached client
	client2, err := integrations.Alibaba(integrations.ClientArgs{})
	s.NoError(err)
	s.NotNil(client2)

	// Both should be the same instance
	s.Equal(client1, client2, "should return cached client instance")
}

func (s *AlibabaNoopSuite) Test_DefaultAlibabaSDK_IsNil() {
	s.Nil(integrations.DefaultAlibabaSDK, "DefaultAlibabaSDK should be nil in noop implementation")
}

func TestAlibabaNoopClient(t *testing.T) {
	suite.Run(t, &AlibabaNoopSuite{})
}
