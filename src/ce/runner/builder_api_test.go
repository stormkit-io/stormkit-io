package runner_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type APIBuilderSuite struct {
	suite.Suite
	config  runner.RunnerOpts
	mockCmd *mocks.CommandInterface
}

func (s *APIBuilderSuite) BeforeTest(_, _ string) {
	tmpDir := path.Join(os.TempDir(), "tmp-test-api-builder-")

	s.config = runner.RunnerOpts{
		RootDir:  tmpDir,
		Reporter: runner.NewReporter("https://example.com"),
		Repo: runner.RepoOpts{
			Dir: path.Join(tmpDir, "repo"),
		},
		Build: runner.BuildOpts{
			EnvVarsRaw: []string{
				"CI=true",
				fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
				fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
			},
		},
	}

	s.NoError(s.config.MkdirAll())

	s.mockCmd = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCmd
}

func (s *APIBuilderSuite) AfterTest(_, _ string) {
	if strings.Contains(s.config.RootDir, os.TempDir()) {
		s.config.RemoveAll()
	}

	s.config.Reporter.Close(nil, nil, nil)
	sys.DefaultCommand = nil
}

func (s *APIBuilderSuite) Test_NewAPIBuilder() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "npm",
	}

	s.NotNil(runner.NewAPIBuilder(ctx, options))
}

func (s *APIBuilderSuite) Test_BuildAll_NoAPIFiles() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "npm",
	}

	// Create empty API directory
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "api"), 0755))

	// It should not fail when there are no API files
	s.NoError(runner.NewAPIBuilder(ctx, options).BuildAll())
}

func (s *APIBuilderSuite) Test_BuildAll_WithJSFiles() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "npm",
	}

	// Create API directory and files
	apiDir := path.Join(s.config.Repo.Dir, "api")
	s.NoError(os.MkdirAll(apiDir, 0755))

	// Create a simple JS API file
	helloAPIContent := `
		export default function handler(req, res) {
  			res.json({ message: 'Hello World!' });
		}
	`

	s.NoError(os.WriteFile(path.Join(apiDir, "hello.js"), []byte(helloAPIContent), 0644))

	// Create a TypeScript API file
	usersAPIContent := `
		interface User {
			id: string;
			name: string;
		}

		export default function handler(req: any, res: any) {
			const users: User[] = [];
			res.json({ users });
		}
	`

	s.NoError(os.WriteFile(path.Join(apiDir, "users.ts"), []byte(usersAPIContent), 0644))

	builder := runner.NewAPIBuilder(ctx, options)
	err := builder.BuildAll()

	s.NoError(err)

	// Check that output files were created
	outputDir := path.Join(s.config.Repo.Dir, "dist")
	s.FileExists(path.Join(outputDir, "hello.mjs"))
	s.FileExists(path.Join(outputDir, "users.mjs"))
}

func (s *APIBuilderSuite) Test_BuildAll_WithNestedDirectories() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "npm",
	}

	// Create nested API structure
	apiDir := path.Join(s.config.Repo.Dir, "api")
	s.NoError(os.MkdirAll(path.Join(apiDir, "users"), 0755))
	s.NoError(os.MkdirAll(path.Join(apiDir, "posts"), 0755))

	// Create nested API files
	getUserContent := `export default (req, res) => res.json({ user: { id: req.params.id } });`
	s.NoError(os.WriteFile(path.Join(apiDir, "users", "[id].js"), []byte(getUserContent), 0644))

	getPostContent := `export default (req, res) => res.json({ post: { id: req.params.id } });`
	s.NoError(os.WriteFile(path.Join(apiDir, "posts", "[id].js"), []byte(getPostContent), 0644))

	builder := runner.NewAPIBuilder(ctx, options)
	err := builder.BuildAll()

	s.NoError(err)

	// Check that nested output files were created
	outputDir := path.Join(s.config.Repo.Dir, "dist")
	s.FileExists(path.Join(outputDir, "users/[id].mjs"))
	s.FileExists(path.Join(outputDir, "posts/[id].mjs"))
}

func (s *APIBuilderSuite) Test_BuildAll_SkipsPrivateFiles() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "npm",
	}

	// Create API directory and files
	apiDir := path.Join(s.config.Repo.Dir, "api")
	s.NoError(os.MkdirAll(apiDir, 0755))

	// Create public file
	s.NoError(os.WriteFile(path.Join(apiDir, "public.js"), []byte(`export default () => 'public';`), 0644))

	// Create private files (should be skipped)
	s.NoError(os.WriteFile(path.Join(apiDir, "_private.js"), []byte(`export default () => 'private';`), 0644))
	s.NoError(os.WriteFile(path.Join(apiDir, "test.spec.js"), []byte(`export default () => 'test';`), 0644))
	s.NoError(os.WriteFile(path.Join(apiDir, "utils.test.js"), []byte(`export default () => 'test';`), 0644))

	builder := runner.NewAPIBuilder(ctx, options)
	err := builder.BuildAll()

	s.NoError(err)

	// Check that only public file was bundled
	outputDir := path.Join(s.config.Repo.Dir, "dist")
	s.FileExists(path.Join(outputDir, "public.mjs"))
	s.NoFileExists(path.Join(outputDir, "_private.mjs"))
	s.NoFileExists(path.Join(outputDir, "test.spec.mjs"))
	s.NoFileExists(path.Join(outputDir, "utils.test.mjs"))
}

func (s *APIBuilderSuite) Test_InstallDependencies_NPM() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		Reporter:       s.config.Reporter,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "npm",
	}

	// Create API directory with package.json
	apiDir := path.Join(s.config.Repo.Dir, "api")
	s.NoError(os.MkdirAll(apiDir, 0755))

	packageJSON := `{
		"name": "api-functions",
		"dependencies": {
			"axios": "^1.0.0"
		}
	}`
	s.NoError(os.WriteFile(path.Join(apiDir, "package.json"), []byte(packageJSON), 0644))

	// Mock npm install command
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Env:    s.config.Build.EnvVarsRaw,
		Name:   "npm",
		Args:   []string{"install"},
		Dir:    apiDir,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil).Once()

	builder := runner.NewAPIBuilder(ctx, options)
	err := builder.InstallDependencies()

	s.NoError(err)
}

func (s *APIBuilderSuite) Test_InstallDependencies_NPM_WithLockFile() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		Reporter:       s.config.Reporter,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "npm",
	}

	// Create API directory with package.json and package-lock.json
	apiDir := path.Join(s.config.Repo.Dir, "api")
	s.NoError(os.MkdirAll(apiDir, 0755))

	packageJSON := `{
		"name": "api-functions",
		"dependencies": {
			"axios": "^1.0.0"
		}
	}`

	s.NoError(os.WriteFile(path.Join(apiDir, "package.json"), []byte(packageJSON), 0644))
	s.NoError(os.WriteFile(path.Join(apiDir, "package-lock.json"), []byte("{}"), 0644))

	// Mock npm ci command (used when package-lock.json exists)
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Env:    s.config.Build.EnvVarsRaw,
		Name:   "npm",
		Args:   []string{"ci", "--include=dev"},
		Dir:    apiDir,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil).Once()

	builder := runner.NewAPIBuilder(ctx, options)
	err := builder.InstallDependencies()

	s.NoError(err)
}

func (s *APIBuilderSuite) Test_InstallDependencies_Yarn() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		Reporter:       s.config.Reporter,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "yarn",
	}

	// Create API directory with package.json
	apiDir := path.Join(s.config.Repo.Dir, "api")
	s.NoError(os.MkdirAll(apiDir, 0755))

	packageJSON := `{
		"name": "api-functions",
		"dependencies": {
			"lodash": "^4.0.0"
		}
	}`

	s.NoError(os.WriteFile(path.Join(apiDir, "package.json"), []byte(packageJSON), 0644))

	// Mock yarn install command
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Env:    s.config.Build.EnvVarsRaw,
		Name:   "yarn",
		Args:   []string{"install"},
		Dir:    apiDir,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil).Once()

	builder := runner.NewAPIBuilder(ctx, options)
	err := builder.InstallDependencies()

	s.NoError(err)
}

func (s *APIBuilderSuite) Test_InstallDependencies_MultiplePackageFiles() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		Reporter:       s.config.Reporter,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "npm",
	}

	// Create nested structure with multiple package.json files
	apiDir := path.Join(s.config.Repo.Dir, "api")
	usersDir := path.Join(apiDir, "users")
	postsDir := path.Join(apiDir, "posts")

	s.NoError(os.MkdirAll(usersDir, 0755))
	s.NoError(os.MkdirAll(postsDir, 0755))

	// Create multiple package.json files
	packageJSON1 := `{"name": "users-api", "dependencies": {"uuid": "^9.0.0"}}`
	packageJSON2 := `{"name": "posts-api", "dependencies": {"moment": "^2.0.0"}}`

	s.NoError(os.WriteFile(path.Join(usersDir, "package.json"), []byte(packageJSON1), 0644))
	s.NoError(os.WriteFile(path.Join(postsDir, "package.json"), []byte(packageJSON2), 0644))

	// Mock npm install commands for both directories
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Env:    s.config.Build.EnvVarsRaw,
		Name:   "npm",
		Args:   []string{"install"},
		Dir:    usersDir,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Env:    s.config.Build.EnvVarsRaw,
		Name:   "npm",
		Args:   []string{"install"},
		Dir:    postsDir,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil).Times(2)

	builder := runner.NewAPIBuilder(ctx, options)
	err := builder.InstallDependencies()

	s.NoError(err)
}

func (s *APIBuilderSuite) Test_BuildAll_SupportedFileExtensions() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		Reporter:       s.config.Reporter,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "npm",
	}

	// Create API directory
	apiDir := path.Join(s.config.Repo.Dir, "api")
	s.NoError(os.MkdirAll(apiDir, 0755))

	// Create files with different supported extensions
	extensions := []string{".js", ".cjs", ".ts", ".mjs", ".tsx", ".jsx"}
	content := `export default () => 'test';`

	for i, ext := range extensions {
		filename := fmt.Sprintf("test%d%s", i, ext)
		s.NoError(os.WriteFile(path.Join(apiDir, filename), []byte(content), 0644))
	}

	// Create unsupported file (should be ignored)
	s.NoError(os.WriteFile(path.Join(apiDir, "readme.txt"), []byte("readme"), 0644))

	builder := runner.NewAPIBuilder(ctx, options)
	err := builder.BuildAll()

	s.NoError(err)

	// Check that supported files were bundled
	outputDir := path.Join(s.config.Repo.Dir, "dist")

	for i := range extensions {
		expectedFile := fmt.Sprintf("test%d.mjs", i)
		s.FileExists(path.Join(outputDir, expectedFile))
	}

	// Check that unsupported file was not bundled
	s.NoFileExists(path.Join(outputDir, "readme.mjs"))
}

func (s *APIBuilderSuite) Test_InstallDependencies_CommandFailure() {
	ctx := context.Background()
	options := runner.APIBuilderOpts{
		WorkDir:        s.config.Repo.Dir,
		EnvVarsSlice:   s.config.Build.EnvVarsRaw,
		Reporter:       s.config.Reporter,
		APIDir:         "api",
		OutputDir:      "dist",
		PackageManager: "npm",
	}

	// Create API directory with package.json
	apiDir := path.Join(s.config.Repo.Dir, "api")
	s.NoError(os.MkdirAll(apiDir, 0755))

	packageJSON := `{"name": "api-functions", "dependencies": {"nonexistent-package": "^999.0.0"}}`
	s.NoError(os.WriteFile(path.Join(apiDir, "package.json"), []byte(packageJSON), 0644))

	// Mock failing npm install command
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Env:    s.config.Build.EnvVarsRaw,
		Name:   "npm",
		Args:   []string{"install"},
		Dir:    apiDir,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(fmt.Errorf("npm install failed")).Once()

	builder := runner.NewAPIBuilder(ctx, options)
	err := builder.InstallDependencies()

	s.Error(err)
	s.Contains(err.Error(), "npm install failed")
}

func TestAPIBuilderSuite(t *testing.T) {
	suite.Run(t, &APIBuilderSuite{})
}
