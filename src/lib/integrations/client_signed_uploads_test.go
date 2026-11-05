package integrations_test

import (
	"context"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
)

// signUploads signs the upload forms of deployment 7 of app 3.
func (s *HCloudSuite) signUploads() *integrations.SignedUploads {
	uploads, err := s.client().SignUploads(context.Background(), integrations.SignUploadsParams{
		AppID:        3,
		DeploymentID: 7,
	})

	s.Require().NoError(err)

	return uploads
}

// Test_SignUploads_Expiry verifies that the forms expire shortly after the
// build.
func (s *HCloudSuite) Test_SignUploads_Expiry() {
	s.LessOrEqual(integrations.SignedUploadsTTL, 10*time.Minute)
}

// Test_SignUploads_ScopedToDeployment verifies that every artifact link points
// at one object under the deployment's own prefix.
func (s *HCloudSuite) Test_SignUploads_ScopedToDeployment() {
	uploads := s.signUploads()

	s.Len(uploads.Artifacts, 4)

	for name, artifact := range uploads.Artifacts {
		link, err := url.Parse(artifact.URL)
		s.Require().NoError(err)

		s.Equal("/artifacts", link.Path)
		s.Equal("3/7/"+name, artifact.Fields["key"])
		s.NotEmpty(artifact.Fields["X-Amz-Signature"])
		s.Equal(int64(1<<30), artifact.MaxSize)
		s.Equal("hcloud:artifacts/3/7/"+name, artifact.Location)
	}
}

// Test_SignUploads_MaxArtifactSize verifies that the instance can change the
// size limit of deployment zips.
func (s *HCloudSuite) Test_SignUploads_MaxArtifactSize() {
	s.T().Setenv(config.MaxArtifactSizeEnvVar, "5")

	uploads := s.signUploads()

	s.Equal(int64(5<<20), uploads.Artifacts["sk-client.zip"].MaxSize)
}

// Test_SignedUploads_TooLarge verifies that a zip over the limit fails with
// the limit and how to change it, before anything is uploaded.
func (s *HCloudSuite) Test_SignedUploads_TooLarge() {
	uploads := s.signUploads()
	artifact := uploads.Artifacts["sk-client.zip"]
	artifact.MaxSize = 1
	uploads.Artifacts["sk-client.zip"] = artifact

	_, err := uploads.Upload(integrations.UploadArgs{ClientZip: path.Join(s.tmpdir, "sk-client.zip")})

	s.ErrorContains(err, "while the limit is 0MB")
	s.ErrorContains(err, config.MaxArtifactSizeEnvVar)
	s.False(s.storage.has("artifacts/3/7/sk-client.zip"))
}

// Test_SignedUploads_StorageEnforcesLimit verifies that the size limit holds
// even when the build ignores it, since storage checks the signed policy.
func (s *HCloudSuite) Test_SignedUploads_StorageEnforcesLimit() {
	s.T().Setenv(config.MaxArtifactSizeEnvVar, "1")

	uploads := s.signUploads()
	big := path.Join(s.tmpdir, "sk-client.zip")

	s.Require().NoError(os.WriteFile(big, make([]byte, 2<<20), 0664))

	artifact := uploads.Artifacts["sk-client.zip"]
	artifact.MaxSize = 0
	uploads.Artifacts["sk-client.zip"] = artifact

	_, err := uploads.Upload(integrations.UploadArgs{ClientZip: big})

	s.ErrorContains(err, "EntityTooLarge")
	s.False(s.storage.has("artifacts/3/7/sk-client.zip"))
}

// uploadSigned uploads the suite's zips through the links of deployment 7.
func (s *HCloudSuite) uploadSigned(uploads *integrations.SignedUploads) *integrations.UploadResult {
	result, err := uploads.Upload(integrations.UploadArgs{
		ClientZip:     path.Join(s.tmpdir, "sk-client.zip"),
		ServerZip:     path.Join(s.tmpdir, "sk-server.zip"),
		ServerHandler: "server.js:handler",
		MigrationsZip: path.Join(s.tmpdir, "sk-migrations.zip"),
	})

	s.Require().NoError(err)

	return result
}

// verify verifies the uploads of deployment 7.
func (s *HCloudSuite) verify(result *integrations.UploadResult) error {
	return s.client().VerifyUploads(integrations.VerifyUploadsParams{
		AppID:        3,
		DeploymentID: 7,
		Result:       result,
	})
}

// Test_SignedUploads_Upload verifies that a build uploading through the links
// produces a deployment the servers can serve, as with the bucket's keys.
func (s *HCloudSuite) Test_SignedUploads_Upload() {
	result := s.uploadSigned(s.signUploads())

	s.Equal("hcloud:artifacts/3/7/sk-client.zip", result.Client.Location)
	s.Equal("hcloud:artifacts/3/7/sk-server.zip/server.js:handler", result.Server.Location)
	s.Equal("hcloud:artifacts/3/7/sk-migrations.zip", result.Migrations.Location)
	s.Empty(result.API.Location)
	s.Positive(result.Client.BytesUploaded)
	s.NoError(s.verify(result))

	file, err := s.client().GetFile(integrations.GetFileArgs{Location: result.Client.Location, FileName: "index.html"})
	s.NoError(err)
	s.Require().NotNil(file)
	s.Equal("Hello world", string(file.Content))
}

// Test_SignUploads_Lambda verifies that clients publishing functions to Lambda
// sign nothing, so their builds keep using credentials.
func (s *HCloudSuite) Test_SignUploads_Lambda() {
	client, err := integrations.AWS(integrations.ClientArgs{AccessKey: "a", SecretKey: "b", Region: "eu-central-1"}, nil)
	s.Require().NoError(err)

	s.False(client.SignsUploads())

	uploads, err := client.SignUploads(context.Background(), integrations.SignUploadsParams{AppID: 3, DeploymentID: 7})
	s.NoError(err)
	s.Nil(uploads)

	result := &integrations.UploadResult{Client: integrations.UploadOverview{Location: "aws:bucket/1/2/sk-client.zip"}}
	s.NoError(client.VerifyUploads(integrations.VerifyUploadsParams{AppID: 3, DeploymentID: 7, Result: result}))
}

// Test_VerifyUploads_ForeignLocations verifies that only the deployment's own
// zips are accepted.
func (s *HCloudSuite) Test_VerifyUploads_ForeignLocations() {
	for _, location := range []string{
		"hcloud:artifacts/3/70/sk-client.zip",
		"hcloud:artifacts/4/7/sk-client.zip",
		"hcloud:other-bucket/3/7/sk-client.zip",
		"hcloud:artifacts/3/7/other.zip",
		"hcloud:artifacts/3/7/../8/sk-client.zip",
		"hcloud:artifacts/3/7/sk-client.zip/../../8/sk-client.zip",
		"local:/shared/deployments/deployment-1/client",
	} {
		err := s.verify(&integrations.UploadResult{Client: integrations.UploadOverview{Location: location}})
		s.ErrorContains(err, "is not an upload of this deployment", location)
	}
}

// Test_SignedUploads_RetriesServerErrors verifies that a brief storage outage
// does not fail the upload.
func (s *HCloudSuite) Test_SignedUploads_RetriesServerErrors() {
	defer integrations.SetSignedRetryDelay(time.Millisecond)()

	uploads := s.signUploads()

	s.storage.failNext(3)

	s.uploadSigned(uploads)
	s.True(s.storage.has("artifacts/3/7/sk-client.zip"))
}

// Test_SignedUploads_GivesUpOnServerErrors verifies that a lasting outage
// fails the upload with the storage's answer.
func (s *HCloudSuite) Test_SignedUploads_GivesUpOnServerErrors() {
	defer integrations.SetSignedRetryDelay(time.Millisecond)()

	uploads := s.signUploads()

	s.storage.failNext(4)

	_, err := uploads.Upload(integrations.UploadArgs{ClientZip: path.Join(s.tmpdir, "sk-client.zip")})
	s.ErrorContains(err, "503")
}

func (s *HCloudSuite) Test_SignedUploads_UnknownZip() {
	_, err := s.signUploads().Upload(integrations.UploadArgs{
		ClientZip: path.Join(s.tmpdir, "other.zip"),
	})

	s.ErrorContains(err, "no upload link for other.zip")
}

// Test_SignUploads_CloudLimits verifies that Stormkit Cloud signs its own
// deployment size limits, whatever the instance setting.
func (s *HCloudSuite) Test_SignUploads_CloudLimits() {
	config.SetIsStormkitCloud(true)
	defer config.SetIsStormkitCloud(false)

	s.T().Setenv(config.MaxArtifactSizeEnvVar, "5000")

	uploads := s.signUploads()

	s.Equal(integrations.CloudArtifactSizeLimits["sk-client.zip"], uploads.Artifacts["sk-client.zip"].MaxSize)
	s.Equal(integrations.CloudArtifactSizeLimits["sk-server.zip"], uploads.Artifacts["sk-server.zip"].MaxSize)
	s.Equal(int64(5000<<20), uploads.Artifacts["sk-migrations.zip"].MaxSize)
}
