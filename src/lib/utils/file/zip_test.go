package file_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stretchr/testify/suite"
)

type ZipSuite struct {
	suite.Suite
	tmpDir string
}

func (s *ZipSuite) BeforeTest(_, _ string) {
	var err error
	s.tmpDir, err = os.MkdirTemp("", "zip_test")
	s.NoError(err)
}

func (s *ZipSuite) AfterTest(_, _ string) {
	s.NoError(os.RemoveAll(s.tmpDir))
}

func (s *ZipSuite) createZipWithFiles(files map[string][]byte) []byte {
	// Create a buffer to hold the zip content
	var buf bytes.Buffer

	// Create a new zip writer
	zipWriter := zip.NewWriter(&buf)

	for fileName, fileContent := range files {
		zf, err := zipWriter.Create(fileName)
		s.NoError(err)
		_, err = zf.Write(fileContent)
		s.NoError(err)
	}

	// Close the zip writer to finalize the zip content
	s.NoError(zipWriter.Close())

	// Return the zip content as a byte slice
	return buf.Bytes()
}

func (s *ZipSuite) TestUnzip_ValidZipFile() {
	files := map[string][]byte{
		"index.html":         []byte("Hello World"),
		"my/folder/file.txt": []byte("This is a test file."),
	}

	// Create a valid zip file
	zipContent := s.createZipWithFiles(files)
	zipFile := filepath.Join(s.tmpDir, "test.zip")

	s.NoError(os.WriteFile(zipFile, zipContent, 0644))

	// Unzip the file
	s.NoError(os.MkdirAll(filepath.Join(s.tmpDir, "output"), 0755))
	destDir := filepath.Join(s.tmpDir, "output")
	s.NoError(file.Unzip(file.UnzipOpts{zipFile, destDir, false}))

	// Verify the unzipped content
	unzippedFile := filepath.Join(destDir, "index.html")
	content, err := os.ReadFile(unzippedFile)
	s.NoError(err)
	s.Equal([]byte("Hello World"), content)
}

func (s *ZipSuite) TestUnzip_ZipSlipVulnerability() {
	files := map[string][]byte{
		"../index.html": []byte("Hello World"),
	}

	// Create a valid zip file
	zipContent := s.createZipWithFiles(files)
	zipFile := filepath.Join(s.tmpDir, "test-invalid.zip")

	s.NoError(os.WriteFile(zipFile, zipContent, 0644))

	// Unzip the file
	s.NoError(os.MkdirAll(filepath.Join(s.tmpDir, "output"), 0755))
	destDir := filepath.Join(s.tmpDir, "output")
	err := file.Unzip(file.UnzipOpts{zipFile, destDir, false})
	s.Error(err)
}

func (s *ZipSuite) TestZipV2_SingleFile() {
	// Create a test file
	testFile := filepath.Join(s.tmpDir, "test.txt")
	s.NoError(os.WriteFile(testFile, []byte("test content"), 0644))

	// Zip the file
	zipFile := filepath.Join(s.tmpDir, "output.zip")
	err := file.ZipV2(file.ZipArgs{
		Source:     []string{"test.txt"},
		ZipName:    zipFile,
		WorkingDir: s.tmpDir,
	})
	s.NoError(err)

	// Verify the zip file was created and is not empty
	s.False(file.IsZipEmpty(zipFile))
}

func (s *ZipSuite) TestZipV2_Directory_WithParent() {
	// Create a test directory with files
	testDir := filepath.Join(s.tmpDir, "testdir")
	s.NoError(os.MkdirAll(testDir, 0755))
	s.NoError(os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"), 0644))
	s.NoError(os.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("content2"), 0644))

	// Zip the directory with parent
	zipFile := filepath.Join(s.tmpDir, "output.zip")
	err := file.ZipV2(file.ZipArgs{
		Source:        []string{"testdir"},
		ZipName:       zipFile,
		WorkingDir:    s.tmpDir,
		IncludeParent: true,
	})
	s.NoError(err)

	// Verify the zip file was created and is not empty
	s.False(file.IsZipEmpty(zipFile))
}

func (s *ZipSuite) TestZipV2_Directory_WithoutParent() {
	// Create a test directory with files
	testDir := filepath.Join(s.tmpDir, "testdir")
	s.NoError(os.MkdirAll(testDir, 0755))
	s.NoError(os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"), 0644))
	s.NoError(os.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("content2"), 0644))

	// Zip the directory without parent
	zipFile := filepath.Join(s.tmpDir, "output.zip")
	err := file.ZipV2(file.ZipArgs{
		Source:        []string{testDir},
		ZipName:       zipFile,
		WorkingDir:    "",
		IncludeParent: false,
	})
	s.NoError(err)

	// Verify the zip file was created and is not empty
	s.False(file.IsZipEmpty(zipFile))
}

func (s *ZipSuite) TestZipV2_NonExistentFile() {
	// Try to zip a non-existent file - should not error, just skip
	zipFile := filepath.Join(s.tmpDir, "output.zip")
	err := file.ZipV2(file.ZipArgs{
		Source:     []string{"nonexistent.txt"},
		ZipName:    zipFile,
		WorkingDir: s.tmpDir,
	})
	s.NoError(err)
}

func TestZipSuite(t *testing.T) {
	suite.Run(t, &ZipSuite{})
}
