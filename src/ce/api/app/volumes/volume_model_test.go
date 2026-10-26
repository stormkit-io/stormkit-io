package volumes_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type VolumeModelSuite struct {
	suite.Suite
	tmpdir      string
	s3Calls     []string
	deleteInput []*s3.DeleteObjectsInput
}

func (s *VolumeModelSuite) BeforeTest(_, _ string) {
	volumes.CachedAWS = nil
	volumes.CachedFileSys = nil
	integrations.CachedAWSClient = nil
	s.s3Calls = nil
	s.deleteInput = nil
	s.tmpdir = s.T().TempDir()
}

func (s *VolumeModelSuite) AfterTest(_, _ string) {
	volumes.CachedAWS = nil
	volumes.CachedFileSys = nil
	integrations.CachedAWSClient = nil
}

// s3Config returns an S3 volumes config whose client records every S3
// operation and answers it locally instead of reaching AWS.
func (s *VolumeModelSuite) s3Config() *admin.VolumesConfig {
	return &admin.VolumesConfig{
		MountType:  volumes.AWSS3,
		BucketName: "my-bucket",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Finalize.Add(
					middleware.FinalizeMiddlewareFunc("Fake", func(ctx context.Context, fi middleware.FinalizeInput, _ middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
						opName := awsmiddleware.GetOperationName(ctx)
						s.s3Calls = append(s.s3Calls, opName)

						if opName == "DeleteObjects" {
							return middleware.FinalizeOutput{Result: &s3.DeleteObjectsOutput{}}, middleware.Metadata{}, nil
						}

						return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
					}),
					middleware.Before,
				)
			},
			func(stack *middleware.Stack) error {
				return stack.Initialize.Add(
					middleware.InitializeMiddlewareFunc("Capture", func(ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
						if v, ok := in.Parameters.(*s3.DeleteObjectsInput); ok {
							s.deleteInput = append(s.deleteInput, v)
						}

						return next.HandleInitialize(ctx, in)
					}),
					middleware.Before,
				)
			},
		},
	}
}

// legacyFile writes a filesystem volume file the way older builds stored it:
// an absolute path and no mount type in its metadata.
func (s *VolumeModelSuite) legacyFile(id types.ID, name string, write bool) *volumes.File {
	file := &volumes.File{ID: id, Name: name, Path: s.tmpdir}

	if write {
		s.Require().NoError(os.WriteFile(filepath.Join(s.tmpdir, name), []byte("Hello world!"), 0664))
	}

	return file
}

func (s *VolumeModelSuite) Test_MountType_Inferred() {
	s.Equal(volumes.FileSys, (&volumes.File{Path: "/shared/volumes/a1e2"}).MountType())
	s.Equal(volumes.AWSS3, (&volumes.File{Path: "my-bucket/a1e2"}).MountType())
	s.Equal(volumes.AWSS3, (&volumes.File{Path: "/shared", Metadata: utils.Map{"mountType": volumes.AWSS3}}).MountType())
}

// Test_RemoveFiles_LegacyFileSysWithS3Config verifies that filesystem files are
// removed from disk even after the instance switched to S3, without touching S3.
func (s *VolumeModelSuite) Test_RemoveFiles_LegacyFileSysWithS3Config() {
	existing := s.legacyFile(1, "existing.txt", true)
	missing := s.legacyFile(2, "missing.txt", false)

	removed, err := volumes.RemoveFiles(s.s3Config(), []*volumes.File{existing, missing})

	s.NoError(err)
	s.ElementsMatch([]*volumes.File{existing, missing}, removed)
	s.NoFileExists(existing.FullPath())
	s.Empty(s.s3Calls)
}

func (s *VolumeModelSuite) Test_RemoveFiles_Mixed() {
	legacy := s.legacyFile(1, "legacy.txt", true)
	remote := &volumes.File{ID: 2, Name: "remote.txt", Path: "my-bucket/a1e2"}

	removed, err := volumes.RemoveFiles(s.s3Config(), []*volumes.File{legacy, remote})

	s.NoError(err)
	s.ElementsMatch([]*volumes.File{legacy, remote}, removed)
	s.NoFileExists(legacy.FullPath())
	s.Equal([]string{"DeleteObjects"}, s.s3Calls)
	s.Require().Len(s.deleteInput, 1)
	s.Equal("my-bucket", *s.deleteInput[0].Bucket)
	s.Require().Len(s.deleteInput[0].Delete.Objects, 1)
	s.Equal("a1e2/remote.txt", *s.deleteInput[0].Delete.Objects[0].Key)
}

func (s *VolumeModelSuite) Test_RemoveFiles_S3FileWithoutBucket() {
	file := &volumes.File{ID: 7, Name: "remote.txt", Path: "/no-bucket", Metadata: utils.Map{"mountType": volumes.AWSS3}}

	removed, err := volumes.RemoveFiles(s.s3Config(), []*volumes.File{file})

	s.EqualError(err, "volume file 7 has no bucket in its path")
	s.Empty(removed)
	s.Empty(s.s3Calls)
}

// Test_RemoveFiles_S3FileWithFileSysConfig verifies that S3 files fail on their
// own when S3 is not configured, while filesystem files are still removed.
func (s *VolumeModelSuite) Test_RemoveFiles_S3FileWithFileSysConfig() {
	cfg := &admin.VolumesConfig{MountType: volumes.FileSys, RootPath: s.tmpdir}
	legacy := s.legacyFile(1, "legacy.txt", true)
	remote := &volumes.File{ID: 2, Name: "remote.txt", Path: "my-bucket/a1e2"}

	removed, err := volumes.RemoveFiles(cfg, []*volumes.File{legacy, remote})

	s.EqualError(err, "1 volume files are stored in aws:s3 but volumes are not configured for it")
	s.Equal([]*volumes.File{legacy}, removed)
	s.NoFileExists(legacy.FullPath())
}

func (s *VolumeModelSuite) Test_Download_LegacyFileSysWithS3Config() {
	file := s.legacyFile(1, "legacy.txt", true)

	reader, err := volumes.Download(s.s3Config(), file)
	s.Require().NoError(err)
	s.Require().NotNil(reader)

	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	data, err := io.ReadAll(reader)
	s.NoError(err)
	s.Equal("Hello world!", string(data))
	s.Empty(s.s3Calls)
}

func (s *VolumeModelSuite) Test_SanitizeUploadFilename_EmptyString() {
	_, err := volumes.SanitizeUploadFilename("")
	s.EqualError(err, "filename must not be empty")
}

func (s *VolumeModelSuite) Test_SanitizeUploadFilename_WhitespaceOnly() {
	_, err := volumes.SanitizeUploadFilename("   ")
	s.EqualError(err, "filename must not be empty")
}

func (s *VolumeModelSuite) Test_SanitizeUploadFilename_AbsolutePath() {
	_, err := volumes.SanitizeUploadFilename("/etc/passwd")
	s.EqualError(err, "absolute paths are not allowed")
}

func (s *VolumeModelSuite) Test_SanitizeUploadFilename_DotOnly() {
	_, err := volumes.SanitizeUploadFilename(".")
	s.EqualError(err, "invalid filename")
}

func (s *VolumeModelSuite) Test_SanitizeUploadFilename_DotDotOnly() {
	_, err := volumes.SanitizeUploadFilename("..")
	s.EqualError(err, "invalid filename")
}

func (s *VolumeModelSuite) Test_SanitizeUploadFilename_PathTraversal() {
	_, err := volumes.SanitizeUploadFilename("../../etc/passwd")
	s.EqualError(err, "path traversal segments are not allowed")
}

func (s *VolumeModelSuite) Test_SanitizeUploadFilename_PathTraversalInMiddle() {
	_, err := volumes.SanitizeUploadFilename("uploads/../../../etc/passwd")
	s.EqualError(err, "path traversal segments are not allowed")
}

func (s *VolumeModelSuite) Test_SanitizeUploadFilename_ValidSimpleFilename() {
	result, err := volumes.SanitizeUploadFilename("file.txt")
	s.NoError(err)
	s.Equal("file.txt", result)
}

func (s *VolumeModelSuite) Test_SanitizeUploadFilename_ValidNestedPath() {
	result, err := volumes.SanitizeUploadFilename("folder/subfolder/file.txt")
	s.NoError(err)
	s.Equal("folder/subfolder/file.txt", result)
}

func (s *VolumeModelSuite) Test_SanitizeUploadFilename_StripsLeadingTrailingSpaces() {
	result, err := volumes.SanitizeUploadFilename("  file.txt  ")
	s.NoError(err)
	s.Equal("file.txt", result)
}

func TestVolumeModelSuite(t *testing.T) {
	suite.Run(t, &VolumeModelSuite{})
}
