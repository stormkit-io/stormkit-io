package volumes_test

import (
	"bytes"
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
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type VolumeStoreSuite struct {
	suite.Suite
	*factory.Factory
	conn    databasetest.TestDB
	s3Calls []string
}

func (s *VolumeStoreSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.s3Calls = nil
	volumes.CachedAWS = nil
	integrations.CachedAWSClient = nil
}

func (s *VolumeStoreSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	volumes.CachedAWS = nil
	integrations.CachedAWSClient = nil
}

// s3Config returns an S3 volumes config whose client records every S3
// operation and answers it locally instead of reaching AWS.
func (s *VolumeStoreSuite) s3Config() *admin.VolumesConfig {
	return &admin.VolumesConfig{
		MountType:  volumes.AWSS3,
		BucketName: "my-bucket",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Finalize.Add(
					middleware.FinalizeMiddlewareFunc("Fake", func(ctx context.Context, _ middleware.FinalizeInput, _ middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
						opName := awsmiddleware.GetOperationName(ctx)
						s.s3Calls = append(s.s3Calls, opName)

						switch opName {
						case "GetObject":
							return middleware.FinalizeOutput{
								Result: &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader([]byte("from s3")))},
							}, middleware.Metadata{}, nil
						case "DeleteObjects":
							return middleware.FinalizeOutput{Result: &s3.DeleteObjectsOutput{}}, middleware.Metadata{}, nil
						}

						return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
					}),
					middleware.Before,
				)
			},
		},
	}
}

// Test_Insert_ReuploadMovesRowToS3 verifies that re-uploading a disk file after
// switching to S3 points the existing row at S3, so reads and deletes follow it.
func (s *VolumeStoreSuite) Test_Insert_ReuploadMovesRowToS3() {
	ctx := context.Background()
	env := s.MockEnv(nil)
	store := volumes.Store()
	tmpdir := s.T().TempDir()

	s.Require().NoError(os.WriteFile(filepath.Join(tmpdir, "file.txt"), []byte("from disk"), 0664))

	s.Require().NoError(store.Insert(ctx, []*volumes.File{{
		Name:      "file.txt",
		Path:      tmpdir,
		Size:      9,
		CreatedAt: utils.NewUnix(),
	}}, env.ID))

	s.Require().NoError(store.Insert(ctx, []*volumes.File{{
		Name:      "file.txt",
		Path:      "my-bucket/a1e2",
		Size:      7,
		CreatedAt: utils.NewUnix(),
		Metadata:  utils.Map{"mountType": volumes.AWSS3},
	}}, env.ID))

	files, err := store.SelectFiles(ctx, volumes.SelectFilesArgs{EnvID: env.ID})
	s.Require().NoError(err)
	s.Require().Len(files, 1)

	file := files[0]
	s.Equal("my-bucket/a1e2", file.Path)
	s.Equal(int64(7), file.Size)
	s.Equal(utils.Map{"mountType": volumes.AWSS3}, file.Metadata)

	reader, err := volumes.Download(s.s3Config(), file)
	s.Require().NoError(err)
	s.Require().NotNil(reader)

	data, err := io.ReadAll(reader)
	s.NoError(err)
	s.Equal("from s3", string(data))

	removed, err := volumes.RemoveFiles(s.s3Config(), files)
	s.NoError(err)
	s.Equal(files, removed)
	s.Equal([]string{"GetObject", "DeleteObjects"}, s.s3Calls)
}

func (s *VolumeStoreSuite) Test_Insert_LegacyRowWithoutMetadata() {
	ctx := context.Background()
	env := s.MockEnv(nil)
	store := volumes.Store()

	s.Require().NoError(store.Insert(ctx, []*volumes.File{{
		Name:      "file.txt",
		Path:      "/shared/volumes/a1e2",
		CreatedAt: utils.NewUnix(),
	}}, env.ID))

	var isNull bool
	s.Require().NoError(s.conn.QueryRow("SELECT file_metadata IS NULL FROM volumes WHERE env_id = $1", env.ID).Scan(&isNull))
	s.True(isNull)

	files, err := store.SelectFiles(ctx, volumes.SelectFilesArgs{EnvID: env.ID})
	s.Require().NoError(err)
	s.Require().Len(files, 1)
	s.Equal(volumes.FileSys, files[0].MountType())
}

func TestVolumeStoreSuite(t *testing.T) {
	suite.Run(t, &VolumeStoreSuite{})
}
