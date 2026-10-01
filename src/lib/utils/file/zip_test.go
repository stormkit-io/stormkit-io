package file_test

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func (s *ZipSuite) createFilesToBeZipped() {
	dirName := "migrations"
	// Create test directory with mixed files
	testDir := filepath.Join(s.tmpDir, dirName)

	// Create directories (including nested)
	s.NoError(os.MkdirAll(filepath.Join(testDir, "nested", "2024"), 0755))

	// Create SQL files
	s.NoError(os.WriteFile(filepath.Join(testDir, "001_create_users.sql"), []byte("CREATE TABLE users"), 0644))
	s.NoError(os.WriteFile(filepath.Join(testDir, "002_create_posts.sql"), []byte("CREATE TABLE posts"), 0644))

	// Create SQL files in nested directories
	s.NoError(os.WriteFile(filepath.Join(testDir, "nested", "init.sql"), []byte("INIT"), 0644))
	s.NoError(os.WriteFile(filepath.Join(testDir, "nested", "2024", "001.sql"), []byte("2024"), 0644))

	// Create non-SQL files (should be excluded)
	s.NoError(os.WriteFile(filepath.Join(testDir, "README.md"), []byte("# Migrations"), 0644))
	s.NoError(os.WriteFile(filepath.Join(testDir, "config.json"), []byte("{}"), 0644))
}

func (s *ZipSuite) listFilesInZip(zipFile string) map[string]bool {
	r, err := zip.OpenReader(zipFile)
	s.NoError(err)
	defer r.Close()

	fileNames := make(map[string]bool)

	for _, f := range r.File {
		fileNames[f.Name] = true
	}

	return fileNames
}

func (s *ZipSuite) Test_Unzip_ValidZipFile() {
	files := map[string][]byte{
		"index.html":         []byte("Hello World"),
		"my/folder/file.txt": []byte("This is a test file."),
	}

	// Create a valid zip file
	zipContent, err := file.ZipInMemory(files)
	s.NoError(err)

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

func (s *ZipSuite) Test_Unzip_ZipSlipVulnerability() {
	files := map[string][]byte{
		"../index.html": []byte("Hello World"),
	}

	// Create a valid zip file
	zipContent, err := file.ZipInMemory(files)
	s.NoError(err)

	zipFile := filepath.Join(s.tmpDir, "test-invalid.zip")

	s.NoError(os.WriteFile(zipFile, zipContent, 0644))

	// Unzip the file
	s.NoError(os.MkdirAll(filepath.Join(s.tmpDir, "output"), 0755))
	s.Error(file.Unzip(file.UnzipOpts{zipFile, filepath.Join(s.tmpDir, "output"), false}))
}

func (s *ZipSuite) Test_ZipV2_WithGlobPattern_SQLFiles() {
	s.createFilesToBeZipped()

	zipFile := filepath.Join(s.tmpDir, "migrations.zip")

	err := file.ZipV2(file.ZipArgs{
		Source:      []string{"migrations"},
		ZipName:     zipFile,
		WorkingDir:  s.tmpDir,
		GlobPattern: "*.sql",
	})

	s.NoError(err)

	fileNames := s.listFilesInZip(zipFile)

	// Should include SQL files
	s.True(fileNames["001_create_users.sql"], "Should include SQL file")
	s.True(fileNames["002_create_posts.sql"], "Should include SQL file")
	s.True(fileNames["nested/init.sql"], "Should include nested SQL file")
	s.True(fileNames["nested/2024/001.sql"], "Should include nested SQL file")

	// Should NOT include non-SQL files
	s.False(fileNames["README.md"], "Should not include non-SQL file")
	s.False(fileNames["config.json"], "Should not include non-SQL file")
}

func (s *ZipSuite) Test_ZipV2_WithoutGlobPattern_AllFiles() {
	s.createFilesToBeZipped()

	// Create zip WITHOUT glob pattern (should include all files)
	zipFile := filepath.Join(s.tmpDir, "all.zip")

	err := file.ZipV2(file.ZipArgs{
		Source:     []string{"migrations"},
		ZipName:    zipFile,
		WorkingDir: s.tmpDir,
	})

	s.NoError(err)
	s.Equal(6, len(s.listFilesInZip(zipFile)), "Should include all files when no glob pattern is specified")
}

func (s *ZipSuite) Test_ZipIterator() {
	// Create a zip file with multiple files
	files := map[string][]byte{
		"001_init.sql":         []byte("CREATE DATABASE test;"),
		"002_create_users.sql": []byte("CREATE TABLE users (id INT);"),
		"003_add_index.sql":    []byte("CREATE INDEX idx_users ON users(id);"),
	}

	zipContent, err := file.ZipInMemory(files)
	s.NoError(err)

	// Test iteration
	var collectedFiles []string
	var collectedContent []string

	err = file.ZipIterator(zipContent, func(fileName string, content []byte) error {
		collectedFiles = append(collectedFiles, fileName)
		collectedContent = append(collectedContent, string(content))
		return nil
	})

	s.NoError(err)

	// Verify all files were iterated
	s.Equal(3, len(collectedFiles), "Should iterate over all files")

	// Verify files are in sorted order
	s.Equal("001_init.sql", collectedFiles[0])
	s.Equal("002_create_users.sql", collectedFiles[1])
	s.Equal("003_add_index.sql", collectedFiles[2])

	// Verify content
	s.Equal("CREATE DATABASE test;", collectedContent[0])
	s.Equal("CREATE TABLE users (id INT);", collectedContent[1])
	s.Equal("CREATE INDEX idx_users ON users(id);", collectedContent[2])
}

func (s *ZipSuite) Test_ZipIterator_EarlyExit() {
	// Create a zip file with multiple files
	files := map[string][]byte{
		"001_init.sql":         []byte("CREATE DATABASE test;"),
		"002_create_users.sql": []byte("CREATE TABLE users (id INT);"),
		"003_add_index.sql":    []byte("CREATE INDEX idx_users ON users(id);"),
	}

	zipContent, err := file.ZipInMemory(files)
	s.NoError(err)

	// Test early exit by returning false after first file
	var collectedFiles []string

	err = file.ZipIterator(zipContent, func(fileName string, content []byte) error {
		collectedFiles = append(collectedFiles, fileName)
		return fmt.Errorf("stop after first file")
	})

	s.Error(err, "Expected error to stop iteration")

	// Verify only first file was processed
	s.Equal(1, len(collectedFiles), "Should stop after first file when returning false")
	s.Equal("001_init.sql", collectedFiles[0])
}

func (s *ZipSuite) Test_ZipV2_WithGlobPattern_IncludeParent() {
	s.createFilesToBeZipped()

	// Create zip with IncludeParent and glob pattern
	zipFile := filepath.Join(s.tmpDir, "parent.zip")

	err := file.ZipV2(file.ZipArgs{
		Source:        []string{"migrations"},
		ZipName:       zipFile,
		WorkingDir:    s.tmpDir,
		IncludeParent: true,
		GlobPattern:   "*.sql",
	})
	s.NoError(err)

	fileNames := s.listFilesInZip(zipFile)

	for name := range fileNames {
		s.True(strings.HasPrefix(name, "migrations/"), "All files should be under parent directory 'migrations/'")
	}
}

func (s *ZipSuite) createRepoWithVCSDirs() {
	root := filepath.Join(s.tmpDir, "repo")

	s.NoError(os.MkdirAll(filepath.Join(root, ".git"), 0755))
	s.NoError(os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[remote]"), 0644))
	s.NoError(os.MkdirAll(filepath.Join(root, ".idea"), 0755))
	s.NoError(os.WriteFile(filepath.Join(root, ".idea", "workspace.xml"), []byte("x"), 0644))
	s.NoError(os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0755))
	s.NoError(os.WriteFile(filepath.Join(root, ".github", "workflows", "ci.yml"), []byte("y"), 0644))
	s.NoError(os.WriteFile(filepath.Join(root, "index.html"), []byte("hello"), 0644))
	s.NoError(os.MkdirAll(filepath.Join(root, "static"), 0755))
	s.NoError(os.WriteFile(filepath.Join(root, "static", "app.js"), []byte("app"), 0644))
}

func (s *ZipSuite) assertNoVCSDirs(fileNames map[string]bool, prefix string) {
	for name := range fileNames {
		s.False(strings.Contains(name, ".git/"), "zip must not include .git: %s", name)
		s.False(strings.Contains(name, ".idea/"), "zip must not include .idea: %s", name)
		s.False(strings.Contains(name, ".github/"), "zip must not include .github: %s", name)
	}

	s.True(fileNames[prefix+"index.html"], "zip must include index.html")
	s.True(fileNames[prefix+"static/app.js"], "zip must include nested content")
}

// Test_ZipV2_ExcludesVCSDirs_WithoutParent covers the whole-repo client bundle
// path: find-based, no parent. This is the path that leaked .git to the CDN.
func (s *ZipSuite) Test_ZipV2_ExcludesVCSDirs_WithoutParent() {
	s.createRepoWithVCSDirs()

	zipFile := filepath.Join(s.tmpDir, "client.zip")

	err := file.ZipV2(file.ZipArgs{
		Source:        []string{"repo"},
		ZipName:       zipFile,
		WorkingDir:    s.tmpDir,
		IncludeParent: false,
		Exclude:       []string{".git", ".idea", ".github"},
	})

	s.NoError(err)
	s.assertNoVCSDirs(s.listFilesInZip(zipFile), "")
}

// Test_ZipV2_ExcludesVCSDirs_IncludeParent covers the whole-repo server bundle
// path (e.g. a start-command runtime), which zips "." with the parent included.
func (s *ZipSuite) Test_ZipV2_ExcludesVCSDirs_IncludeParent() {
	s.createRepoWithVCSDirs()

	zipFile := filepath.Join(s.tmpDir, "server.zip")

	err := file.ZipV2(file.ZipArgs{
		Source:        []string{"repo"},
		ZipName:       zipFile,
		WorkingDir:    s.tmpDir,
		IncludeParent: true,
		Exclude:       []string{".git", ".idea", ".github"},
	})

	s.NoError(err)
	s.assertNoVCSDirs(s.listFilesInZip(zipFile), "repo/")
}

// Test_ZipV2_NoExclude_KeepsDotDirs verifies the utility stays generic: with no
// Exclude, dot-directories are zipped like any other content.
func (s *ZipSuite) Test_ZipV2_NoExclude_KeepsDotDirs() {
	s.createRepoWithVCSDirs()

	zipFile := filepath.Join(s.tmpDir, "all.zip")

	err := file.ZipV2(file.ZipArgs{
		Source:     []string{"repo"},
		ZipName:    zipFile,
		WorkingDir: s.tmpDir,
	})

	s.NoError(err)

	fileNames := s.listFilesInZip(zipFile)
	s.True(fileNames[".git/config"], "without Exclude, .git should be kept")
	s.True(fileNames["index.html"], "normal content should be kept")
}

// Test_ZipV2_ExcludesGitFile covers a submodule/worktree checkout where .git is
// a plain file (a "gitdir:" pointer) rather than a directory.
func (s *ZipSuite) Test_ZipV2_ExcludesGitFile() {
	root := filepath.Join(s.tmpDir, "repo")

	s.NoError(os.MkdirAll(root, 0755))
	s.NoError(os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: /elsewhere"), 0644))
	s.NoError(os.WriteFile(filepath.Join(root, "index.html"), []byte("hello"), 0644))

	for _, includeParent := range []bool{true, false} {
		zipFile := filepath.Join(s.tmpDir, "out.zip")
		s.NoError(os.RemoveAll(zipFile))

		err := file.ZipV2(file.ZipArgs{
			Source:        []string{"repo"},
			ZipName:       zipFile,
			WorkingDir:    s.tmpDir,
			IncludeParent: includeParent,
			Exclude:       []string{".git"},
		})

		s.NoError(err)

		for name := range s.listFilesInZip(zipFile) {
			s.NotEqual(".git", filepath.Base(name), "gitfile must be excluded (includeParent=%v)", includeParent)
		}
	}
}

func (s *ZipSuite) Test_ZipV2_RejectsInvalidExclude() {
	s.createRepoWithVCSDirs()

	err := file.ZipV2(file.ZipArgs{
		Source:     []string{"repo"},
		ZipName:    filepath.Join(s.tmpDir, "out.zip"),
		WorkingDir: s.tmpDir,
		Exclude:    []string{"../etc"},
	})

	s.Error(err)
}

func TestZipSuite(t *testing.T) {
	suite.Run(t, &ZipSuite{})
}
