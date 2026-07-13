package volumes_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type VolumesAWSSuite struct {
	suite.Suite
	fileHeader *mocks.FileHeader
}

type MockFile struct {
	*bytes.Reader
	io.Closer
}

func (s *VolumesAWSSuite) SetupSuite() {
	s.fileHeader = &mocks.FileHeader{}
	s.fileHeader.On("Open").Return(&MockFile{Reader: bytes.NewReader([]byte(`Hello world!`)), Closer: io.NopCloser(nil)}, nil)
	s.fileHeader.On("Size").Return(int64(691))
	s.fileHeader.On("Name").Return("my-file.txt")
	utils.NewUnix = factory.MockNewUnix
}

func (s *VolumesAWSSuite) BeforeTest(_, _ string) {
	volumes.CachedAWS = nil
	integrations.CachedAWSClient = nil
}

func (s *VolumesAWSSuite) TearDownSuite() {
	utils.NewUnix = factory.OriginalNewUnix
}

func (s *VolumesAWSSuite) Test_Upload() {
	cfg := &admin.VolumesConfig{
		MountType:  volumes.AWSS3,
		BucketName: "my-bucket",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Initialize.Add(
					middleware.InitializeMiddlewareFunc("Upload", func(ctx context.Context, fi middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
						switch v := fi.Parameters.(type) {
						case *s3.PutObjectInput:
							s.Equal("my-bucket", *v.Bucket)
							s.Equal(int64(691), *v.ContentLength)
						default:
							s.NoError(errors.New("unknown call"))
						}

						return next.HandleInitialize(ctx, fi)
					}),
					middleware.Before,
				)
			},
			func(stack *middleware.Stack) error {
				return stack.Finalize.Add(
					middleware.FinalizeMiddlewareFunc("Upload", func(ctx context.Context, fi middleware.FinalizeInput, fh middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
						opName := awsmiddleware.GetOperationName(ctx)

						if opName == "PutObject" {
							return middleware.FinalizeOutput{
								Result: &s3.PutObjectOutput{},
							}, middleware.Metadata{}, nil
						}

						s.NoError(errors.New("unknown call"))

						return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
					}),
					middleware.Before,
				)
			},
		},
	}

	args := volumes.UploadArgs{
		AppID:      5818,
		EnvID:      29151,
		FileHeader: s.fileHeader,
		ContentDisposition: map[string]string{
			"filename": s.fileHeader.Name(),
		},
	}

	file, err := volumes.Upload(cfg, args)
	s.NoError(err)
	s.NotEmpty(file)
	s.Equal(&volumes.File{
		ID:        0,
		EnvID:     args.EnvID,
		Size:      int64(691),
		Name:      "my-file.txt",
		Path:      "my-bucket/a5818e29151",
		Metadata:  utils.Map{"mountType": volumes.AWSS3},
		CreatedAt: utils.NewUnix(),
	}, file)
}

func (s *VolumesAWSSuite) Test_Download() {
	cfg := &admin.VolumesConfig{
		MountType:  volumes.AWSS3,
		BucketName: "my-bucket",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Initialize.Add(
					middleware.InitializeMiddlewareFunc("GetObject", func(ctx context.Context, fi middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
						switch v := fi.Parameters.(type) {
						case *s3.GetObjectInput:
							s.Equal("my-bucket", *v.Bucket)
							s.Equal("a1e2/my-file.txt", *v.Key)
						default:
							s.NoError(errors.New("unknown call"))
						}

						return next.HandleInitialize(ctx, fi)
					}),
					middleware.Before,
				)
			},
			func(stack *middleware.Stack) error {
				return stack.Finalize.Add(
					middleware.FinalizeMiddlewareFunc("GetObject", func(ctx context.Context, fi middleware.FinalizeInput, fh middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
						opName := awsmiddleware.GetOperationName(ctx)

						if opName == "GetObject" {
							return middleware.FinalizeOutput{
								Result: &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader([]byte("Hello world!")))},
							}, middleware.Metadata{}, nil
						}

						s.NoError(errors.New("unknown call"))

						return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
					}),
					middleware.Before,
				)
			},
		},
	}

	file, err := volumes.Download(cfg, &volumes.File{Path: "my-bucket/a1e2", Name: "my-file.txt"})
	s.NoError(err)
	s.NotEmpty(file)

	data, err := io.ReadAll(file)
	s.NoError(err)
	s.Equal("Hello world!", string(data))
}

func (s *VolumesAWSSuite) Test_RemoveFiles() {
	cfg := &admin.VolumesConfig{
		MountType:  volumes.AWSS3,
		BucketName: "my-bucket",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Initialize.Add(
					middleware.InitializeMiddlewareFunc("DeleteObjects", func(ctx context.Context, fi middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
						switch v := fi.Parameters.(type) {
						case *s3.DeleteObjectsInput:
							s.Equal("my-bucket", *v.Bucket)
							s.Equal(types.Delete{
								Quiet: aws.Bool(true),
								Objects: []types.ObjectIdentifier{
									{Key: aws.String("a1e2/my-file.txt")},
								}}, *v.Delete)
						default:
							s.NoError(errors.New("unknown call"))
						}

						return next.HandleInitialize(ctx, fi)
					}),
					middleware.Before,
				)
			},
			func(stack *middleware.Stack) error {
				return stack.Finalize.Add(
					middleware.FinalizeMiddlewareFunc("DeleteObjects", func(ctx context.Context, fi middleware.FinalizeInput, fh middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
						opName := awsmiddleware.GetOperationName(ctx)

						if opName == "DeleteObjects" {
							return middleware.FinalizeOutput{
								Result: &s3.DeleteObjectsOutput{},
							}, middleware.Metadata{}, nil
						}

						s.NoError(errors.New("unknown call"))

						return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
					}),
					middleware.Before,
				)
			},
		},
	}

	files, err := volumes.RemoveFiles(cfg, []*volumes.File{{Path: "my-bucket/a1e2", Name: "my-file.txt"}})
	s.NoError(err)
	s.NotEmpty(files)
}

func TestVolumesAWSSuite(t *testing.T) {
	suite.Run(t, &VolumesAWSSuite{})
}
