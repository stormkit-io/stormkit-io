package volumes_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stretchr/testify/suite"
)

type VolumeModelSuite struct {
	suite.Suite
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
