package integrations_test

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stretchr/testify/suite"
)

type ZipManagerSuite struct {
	suite.Suite

	tmpdir string
}

func (s *ZipManagerSuite) BeforeTest(suiteName, _ string) {
	var err error
	s.tmpdir, err = os.MkdirTemp("", "tmp-integrations-zip-manager-")
	s.NoError(err)
}

func (s *ZipManagerSuite) AfterTest(_, _ string) {
	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}
}

func (s *ZipManagerSuite) createFiles() string {
	clientDir := path.Join(s.tmpdir, "client")

	s.NoError(os.MkdirAll(clientDir, 0774))
	s.NoError(os.WriteFile(path.Join(clientDir, "index.html"), []byte("Hello world"), 0664))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(s.tmpdir, "sk-client.zip")}))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(s.tmpdir, "sk-server.zip")}))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(s.tmpdir, "sk-api.zip")}))

	return clientDir
}

func (s *ZipManagerSuite) Test_Download() {
	called := 0

	zipManager := integrations.NewZipManager(func(deploymentID, bucketname, keyprefix string) (string, error) {
		called = called + 1
		return s.createFiles(), nil
	})

	test := func() {
		file, err := zipManager.GetFile(integrations.GetFileArgs{
			Location:     "my-bucket/my-key-prefix",
			FileName:     "/index.html",
			DeploymentID: types.ID(10),
		})

		s.NoError(err)
		s.Equal("Hello world", string(file.Content))
		s.Equal(int64(len("Hello world")), file.Size)
	}

	for i := 0; i < 5; i = i + 1 {
		test()
	}

	s.Equal(1, called)

	// Removing the tmp dir should call the download function once again
	s.NoError(os.RemoveAll(path.Join(s.tmpdir, "client")))

	test()

	s.Equal(2, called)
}

func (s *ZipManagerSuite) Test_Download_NotFound() {
	zipManager := integrations.NewZipManager(func(deploymentID, bucketname, keyprefix string) (string, error) {
		s.createFiles()
		return path.Join(s.tmpdir, "client"), nil
	})

	file, err := zipManager.GetFile(integrations.GetFileArgs{
		Location:     "my-bucket/my-key-prefix",
		FileName:     "/file-not-found.html",
		DeploymentID: types.ID(10),
	})

	s.Error(err)
	s.Nil(file)
}

// Test_PathTraversal_Rejected verifies that a fileName containing ".."
// segments cannot escape the deployment directory.
func (s *ZipManagerSuite) Test_PathTraversal_Rejected() {
	zipManager := integrations.NewZipManager(func(deploymentID, bucketname, keyprefix string) (string, error) {
		return s.createFiles(), nil
	})

	_, err := zipManager.GetFile(integrations.GetFileArgs{
		Location:     "my-bucket/my-key-prefix",
		FileName:     "/index.html",
		DeploymentID: types.ID(10),
	})
	s.NoError(err)

	f, err := zipManager.GetFile(integrations.GetFileArgs{
		Location:     "my-bucket/my-key-prefix",
		FileName:     "../../../etc/passwd",
		DeploymentID: types.ID(10),
	})
	s.Error(err)
	s.Nil(f)
}

func TestZipManager(t *testing.T) {
	suite.Run(t, &ZipManagerSuite{})
}
