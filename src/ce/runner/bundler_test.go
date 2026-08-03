package runner_test

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type BundlerSuite struct {
	suite.Suite
	config runner.RunnerOpts
}

func (s *BundlerSuite) BeforeTest(_, _ string) {
	tmpDir := path.Join(os.TempDir(), "tmp-test-runner-")
	repoDir := path.Join(tmpDir, "repo")

	s.config = runner.RunnerOpts{
		RootDir:     tmpDir,
		WorkDir:     repoDir,
		Reporter:    runner.NewReporter("https://example.com"),
		AutoInstall: true,
		Repo: runner.RepoOpts{
			Dir: repoDir,
		},
		Build: runner.BuildOpts{
			MigrationsFolder: "/migrations",
		},
	}

	s.NoError(s.config.MkdirAll())
}

func (s *BundlerSuite) AfterTest(_, _ string) {
	if strings.Contains(s.config.RootDir, os.TempDir()) {
		s.config.RemoveAll()
	}

	s.config.Reporter.Close(nil, nil, nil)

	runner.APIWrapper = ""
	sys.DefaultCommand = nil
}

func (s *BundlerSuite) Test_Bundle_AllFolder() {
	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.Empty(artifacts.Redirects)
	s.Empty(artifacts.ApiDirs)
	s.Empty(artifacts.ServerDirs)
	s.Empty(artifacts.FunctionHandler)
	s.Empty(artifacts.ApiHandler)
	s.Equal([]string{"."}, artifacts.ClientDirs)
}

func (s *BundlerSuite) Test_Bundle_StormkitFolder_And_PublicFolder() {
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".stormkit", "public"), 0774))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".stormkit", "server"), 0774))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".stormkit", "api"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, ".stormkit", "server", "index.js"), []byte("Hello world"), 0664))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "public"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "public", "favicon.ico"), []byte("Hello world"), 0664))

	text := "Some data that will be injected into stormkit-api.mjs"
	runner.APIWrapper = text

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.Empty(artifacts.Redirects)

	s.Equal("stormkit-api.mjs:handler", artifacts.ApiHandler)
	s.Equal("index.js:handler", artifacts.FunctionHandler)

	s.Equal([]string{".stormkit/api"}, artifacts.ApiDirs)
	s.Equal([]string{".stormkit/server"}, artifacts.ServerDirs)
	s.Equal([]string{"public", ".stormkit/public"}, artifacts.ClientDirs)

	// We should also have a `stormkit-api.mjs` file
	data, err := os.ReadFile(path.Join(s.config.Repo.Dir, ".stormkit", "api", "stormkit-api.mjs"))

	s.NoError(err)
	s.Equal(text, string(data))
}

func (s *BundlerSuite) Test_Bundle_WithDistFolderSpecified() {
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "public"), 0774))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "build", "public"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "build", "index.js"), []byte("Hello world"), 0664))

	s.config.Build.DistFolder = "build"
	s.config.Build.ServerCmd = "npm run start"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)

	s.Empty(artifacts.ApiDirs)
	s.Equal([]string{"public"}, artifacts.ClientDirs)
	s.Equal([]string{"build"}, artifacts.ServerDirs)
	s.Equal(".:server", artifacts.FunctionHandler)
}

func (s *BundlerSuite) Test_Bundle_NextServer() {
	nextPck := `{ 
		"name": "next", 
		"version": "1.0.0", 
		"files": ["index.js", "dist"],
		"scripts": {
			"start": "next start"
		}
	}`

	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".next"), 0755))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, ".next", "index.js"), []byte("Hello world"), 0644))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "node_modules", ".bin"), 0755))
	s.NoError(os.Symlink(path.Join("..", "next", "dist", "bin", "next"), path.Join(s.config.Repo.Dir, "node_modules", ".bin", "next")))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "node_modules", "next", "dist", "bin"), 0755))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "node_modules", "next", "dist", "bin", "next"), []byte("console.log('hi')"), 0755))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "node_modules", "next", "package.json"), []byte(nextPck), 0644))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "node_modules", "next", "index.js"), []byte("Hello world"), 0644))

	packageJson := &runner.PackageJson{
		Name:    "my-next-app",
		Version: "1.0.0-test",
		Dependencies: map[string]string{
			"next": "1.0.0-test",
		},
	}

	s.NoError(packageJson.Write(path.Join(s.config.Repo.Dir, "package.json")))

	s.config.Repo.PackageJson = packageJson
	s.config.Build.ServerCmd = "npm run start"
	s.config.Build.DistFolder = ".next"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.NotNil(artifacts)
	s.Empty(artifacts.ApiDirs)
	s.Empty(artifacts.ClientDirs)
	s.Equal(".:server", artifacts.FunctionHandler)
	s.Equal([]string{"package.json", "node_modules/.bin", ".next"}, artifacts.ServerDirs)
	s.FileExists(path.Join(s.config.Repo.Dir, ".next", "index.js"))
	s.FileExists(path.Join(s.config.Repo.Dir, "node_modules", ".bin", "next")) // This is the symlink
}

func (s *BundlerSuite) Test_Bundle_AllRepo() {
	s.config.Build.ServerCmd = "npm run start"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.NotNil(artifacts)
	s.Empty(artifacts.ApiDirs)
	s.Empty(artifacts.ClientDirs)
	s.Equal(".:server", artifacts.FunctionHandler)
	s.Equal([]string{"."}, artifacts.ServerDirs)
}

func (s *BundlerSuite) Test_Bundle_CopiesFlakeNix_WholeDir() {
	// flake.nix in repoDir, whole workDir bundled
	s.config.Build.ServerCmd = "npm run start"
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "flake.nix"), []byte("{}"), 0664))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "flake.lock"), []byte("{}"), 0664))

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.Equal([]string{"."}, artifacts.ServerDirs)

	// flake.nix was already in workDir — no copy needed, but it must be present
	s.FileExists(path.Join(s.config.Repo.Dir, "flake.nix"))
	s.FileExists(path.Join(s.config.Repo.Dir, "flake.lock"))
}

func (s *BundlerSuite) Test_Bundle_CopiesFlakeNix_ToServerDir() {
	// flake.nix in workDir, server output is a dist subdirectory.
	// flake files must appear at workDir root in ServerDirs so the zip
	// places them at the top level — not nested inside dist/.
	distDir := path.Join(s.config.Repo.Dir, "dist")
	s.NoError(os.MkdirAll(distDir, 0774))
	s.NoError(os.WriteFile(path.Join(distDir, "server"), []byte("binary"), 0775))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "flake.nix"), []byte("{}"), 0664))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "flake.lock"), []byte("{}"), 0664))

	s.config.Build.ServerCmd = "./dist/server"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.FileExists(path.Join(s.config.Repo.Dir, "flake.nix"))
	s.FileExists(path.Join(s.config.Repo.Dir, "flake.lock"))
	s.Contains(artifacts.ServerDirs, "flake.nix")
	s.Contains(artifacts.ServerDirs, "flake.lock")
}

func (s *BundlerSuite) Test_Bundle_CopiesFlakeNix_FromRepoDirToServerDir() {
	// flake.nix is in repoDir but workDir is a subdirectory (SK_CWD case).
	// The file must be copied to workDir root and listed in ServerDirs so
	// it lands at the top level of the zip, not nested inside dist/.
	subDir := path.Join(s.config.Repo.Dir, "app")
	serverDir := path.Join(subDir, "dist")
	s.NoError(os.MkdirAll(serverDir, 0774))
	s.NoError(os.WriteFile(path.Join(serverDir, "server"), []byte("binary"), 0775))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "flake.nix"), []byte("{}"), 0664))

	s.config.WorkDir = subDir
	s.config.Build.DistFolder = "dist"
	s.config.Build.ServerCmd = "./dist/server"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.FileExists(path.Join(subDir, "flake.nix"))
	s.Contains(artifacts.ServerDirs, "flake.nix")
}

// serverZipEntries zips the artifacts and returns the file names inside the
// server zip, so tests can assert the layout the runtime will actually see.
func (s *BundlerSuite) serverZipEntries(bundler runner.BundlerInterface, artifacts *runner.Artifacts) []string {
	// Zip errors out when the configured migrations folder is missing.
	s.Require().NoError(os.MkdirAll(path.Join(s.config.WorkDir, "migrations"), 0774))
	s.Require().NoError(bundler.Zip(artifacts))

	reader, err := zip.OpenReader(path.Join(s.config.RootDir, "dist", "sk-server.zip"))
	s.Require().NoError(err)

	defer reader.Close()

	names := []string{}

	for _, f := range reader.File {
		names = append(names, f.Name)
	}

	return names
}

func (s *BundlerSuite) Test_Bundle_StormkitServerFolder_WithServerCmd() {
	// Application runtime: a start command makes `.stormkit/server` an ordinary
	// server folder, so it is bundled exactly like `dist` or `.next` would be —
	// no serverless entry file detection, and the parent folder is kept.
	serverDir := path.Join(s.config.Repo.Dir, ".stormkit", "server")
	s.NoError(os.MkdirAll(serverDir, 0774))
	s.NoError(os.WriteFile(path.Join(serverDir, "app"), []byte("binary"), 0775))

	s.config.Build.ServerCmd = "./.stormkit/server/app"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.Empty(artifacts.ApiDirs)
	s.Empty(artifacts.ClientDirs)
	s.Equal(".:server", artifacts.FunctionHandler)
	s.Equal([]string{".stormkit/server"}, artifacts.ServerDirs)
	s.Contains(s.serverZipEntries(bundler, artifacts), ".stormkit/server/app")
}

func (s *BundlerSuite) Test_Bundle_StormkitServerFolder_WithServerCmd_KeepsFlakeAtRoot() {
	// The zip is rooted at the working directory, so the flake stays where the
	// generic server-command path puts it instead of being copied into the folder.
	serverDir := path.Join(s.config.Repo.Dir, ".stormkit", "server")
	s.NoError(os.MkdirAll(serverDir, 0774))
	s.NoError(os.WriteFile(path.Join(serverDir, "app"), []byte("binary"), 0775))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "flake.nix"), []byte("{}"), 0664))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "flake.lock"), []byte("{}"), 0664))

	s.config.Build.ServerCmd = "./.stormkit/server/app"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.Contains(artifacts.ServerDirs, ".stormkit/server")
	s.Contains(artifacts.ServerDirs, "flake.nix")
	s.Contains(artifacts.ServerDirs, "flake.lock")
	s.NoFileExists(path.Join(serverDir, "flake.nix"))

	entries := s.serverZipEntries(bundler, artifacts)

	s.Contains(entries, "flake.nix")
	s.Contains(entries, ".stormkit/server/app")
}

func (s *BundlerSuite) Test_Bundle_StormkitServerFolder_WithoutServerCmd_IsFlattened() {
	// Without a start command the folder keeps its serverless meaning: the entry
	// file is auto detected and the zip holds the folder contents at the root.
	serverDir := path.Join(s.config.Repo.Dir, ".stormkit", "server")
	s.NoError(os.MkdirAll(serverDir, 0774))
	s.NoError(os.WriteFile(path.Join(serverDir, "index.js"), []byte("export const handler = () => {}"), 0664))

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.Equal("index.js:handler", artifacts.FunctionHandler)
	s.Equal([]string{".stormkit/server"}, artifacts.ServerDirs)

	entries := s.serverZipEntries(bundler, artifacts)

	s.Contains(entries, "index.js")
	s.NotContains(entries, ".stormkit/server/index.js")
}

func (s *BundlerSuite) Test_Bundle_DistFolder_WinsOver_StormkitServerFolder() {
	// A leftover `.stormkit/server` must not shadow an explicitly configured
	// output folder.
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".stormkit", "server"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, ".stormkit", "server", "leftover"), []byte("stale"), 0664))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "dist"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "dist", "index.js"), []byte("Hello world"), 0664))

	s.config.Build.DistFolder = "dist"
	s.config.Build.ServerCmd = "npm run start"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.Equal([]string{"dist"}, artifacts.ServerDirs)
}

func (s *BundlerSuite) Test_Bundle_DistFolder_Stormkit_KeepsSubfolderLayout() {
	// `.stormkit` is the convention folder, not a custom output folder, so
	// setting it explicitly must still resolve to the `.stormkit/*` subfolders.
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".stormkit", "server"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, ".stormkit", "server", "app"), []byte("binary"), 0775))

	s.config.Build.DistFolder = ".stormkit"
	s.config.Build.ServerCmd = "./.stormkit/server/app"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.Equal([]string{".stormkit/server"}, artifacts.ServerDirs)
}

func (s *BundlerSuite) Test_Bundle_StormkitServerFolder_WithoutServerCmd_RequiresEntryFile() {
	serverDir := path.Join(s.config.Repo.Dir, ".stormkit", "server")
	s.NoError(os.MkdirAll(serverDir, 0774))
	s.NoError(os.WriteFile(path.Join(serverDir, "app"), []byte("binary"), 0775))

	bundler := runner.NewBundler(s.config)
	_, err := bundler.Bundle(context.Background())

	s.ErrorContains(err, "cannot auto detect serverless entry file")
}

func (s *BundlerSuite) Test_Bundle_NextServer_AlternativeSyntax() {
	s.config.Build.ServerCmd = "npm run start"
	s.config.Build.DistFolder = ".next"

	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".next"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, ".next", "index.js"), []byte("Hello world"), 0664))

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)

	s.Empty(artifacts.ApiDirs)
	s.Empty(artifacts.ClientDirs)
	s.Equal(".:server", artifacts.FunctionHandler)
	s.Equal([]string{".next"}, artifacts.ServerDirs)
	s.FileExists(path.Join(s.config.Repo.Dir, ".next", "index.js"))
}

func (s *BundlerSuite) Test_Bundle_BundleDeps_Case_StormkitServerless() {
	ctx := context.Background()
	mockCmd := &mocks.CommandInterface{}
	sys.DefaultCommand = mockCmd

	packageJsonData := `{
		"name": "sample-project",
		"version": "1.0.0",
		"dependencies": {
			"react": "18.2.0",
			"@stormkit/serverless": "2.0.8"
		},
		"bundleDependencies": [
			"@stormkit/serverless"
		]
	}`

	pck := &runner.PackageJson{}

	s.NoError(json.Unmarshal([]byte(packageJsonData), pck))
	s.NoError(os.MkdirAll(path.Join(s.config.WorkDir, ".stormkit", "public"), 0755))
	s.NoError(os.MkdirAll(path.Join(s.config.WorkDir, ".stormkit", "server"), 0755))
	s.NoError(os.WriteFile(path.Join(s.config.WorkDir, ".stormkit", "server", "index.js"), []byte("require('react'); await import('my-dep')"), 0664))
	s.NoError(os.WriteFile(path.Join(s.config.WorkDir, "package.json"), []byte(packageJsonData), 0664))

	s.config.Repo.PackageJson = pck

	files := []string{
		"package.json",
		"node_modules/.bin",
		"node_modules/@stormkit/serverless",
		"node_modules/react",
		// "node_modules/my-dep" should not be included because it doesn't exist in dependencies
	}

	for _, file := range files {
		mockCmd.On("SetOpts", sys.CommandOpts{
			Name: "rsync",
			Args: []string{"-a", "-R", file, ".stormkit/server"},
			Dir:  s.config.Repo.Dir,
		}).Return(mockCmd).Once()

		mockCmd.On("Run").Return(nil).Once()
	}

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(ctx)

	s.NoError(err)
	s.Equal([]string{".stormkit/server"}, artifacts.ServerDirs)
}

func (s *BundlerSuite) Test_Bundle_BundleDeps_Case_Server() {
	ctx := context.Background()
	mockCmd := &mocks.CommandInterface{}
	sys.DefaultCommand = mockCmd

	packageJsonData := `{
		"name": "sample-project",
		"version": "1.0.0",
		"dependencies": {
			"vue": "5.3.4",
			"@stormkit/serverless": "2.0.8"
		},
		"bundleDependencies": [
			"@stormkit/serverless"
		]
	}`

	pck := &runner.PackageJson{}

	s.NoError(json.Unmarshal([]byte(packageJsonData), pck))
	s.NoError(os.MkdirAll(path.Join(s.config.WorkDir, "dist"), 0755))
	s.NoError(os.WriteFile(path.Join(s.config.WorkDir, "dist", "index.js"), []byte("require('vue'); await import('my-dep')"), 0664))
	s.NoError(os.WriteFile(path.Join(s.config.WorkDir, "package.json"), []byte(packageJsonData), 0664))

	s.config.Repo.PackageJson = pck
	s.config.Build.ServerCmd = "npm run start"
	s.config.Build.DistFolder = "dist"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(ctx)

	slices.Sort(artifacts.ServerDirs)

	s.NoError(err)
	s.EqualValues([]string{
		"dist",
		"node_modules/.bin",
		"node_modules/@stormkit/serverless",
		"node_modules/vue",
		"package.json",
		// "node_modules/my-dep" should not be included because it doesn't exist in dependencies
	}, artifacts.ServerDirs)
}

func (s *BundlerSuite) Test_Bundle_CustomDistFolder() {
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "output"), 0774))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".stormkit", "api"), 0774))

	s.config.Build.DistFolder = "output"

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)
	s.Empty(artifacts.Redirects)
	s.Empty(artifacts.FunctionHandler)
	s.Empty(artifacts.ServerDirs)

	s.Equal([]string{"output"}, artifacts.ClientDirs)
	s.Equal([]string{".stormkit/api"}, artifacts.ApiDirs)
	s.Equal("stormkit-api.mjs:handler", artifacts.ApiHandler)
}

func (s *BundlerSuite) Test_Redirects_Stormkit() {
	redirects := []deploy.Redirect{
		{From: "/redirects/permanent", To: "/", Status: 301},
		{From: "/redirects/temporary", To: "/", Status: 302},
		{From: "/redirects/rewrite", To: "/"},
		{From: "/redirects/proxy", To: "https://example.com/"},
	}

	data, err := json.Marshal(redirects)

	s.NoError(err)
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "redirects.json"), data, 0664))

	bundler := runner.NewBundler(s.config)
	artifacts := runner.Artifacts{}

	s.NoError(bundler.ParseRedirects(&artifacts))
	s.Len(artifacts.Redirects, 4)
	s.Equal(redirects[0], artifacts.Redirects[0])
	s.Equal(redirects[1], artifacts.Redirects[1])
	s.Equal(redirects[2], artifacts.Redirects[2])
	s.Equal(redirects[3], artifacts.Redirects[3])
}

func (s *BundlerSuite) Test_Redirects_CustomFile() {
	redirects := []deploy.Redirect{
		{From: "/redirects/permanent", To: "/", Status: 301},
		{From: "/redirects/temporary", To: "/", Status: 302},
		{From: "/redirects/rewrite", To: "/"},
		{From: "/redirects/proxy", To: "https://example.com/"},
	}

	data, err := json.Marshal(redirects)

	s.NoError(err)
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "custom-redirects.json"), data, 0664))

	s.config.Build.RedirectsFile = "custom-redirects.json"

	bundler := runner.NewBundler(s.config)

	artifacts := runner.Artifacts{}

	s.NoError(bundler.ParseRedirects(&artifacts))
	s.Len(artifacts.Redirects, 4)
	s.Equal(redirects[0], artifacts.Redirects[0])
	s.Equal(redirects[1], artifacts.Redirects[1])
	s.Equal(redirects[2], artifacts.Redirects[2])
	s.Equal(redirects[3], artifacts.Redirects[3])
}

func (s *BundlerSuite) Test_Redirects_Netlify() {
	data := `
		/home                /
		/blog/my-post.php    /blog/my-post 			302!
		/news/*              /blog/:splat			307
		/cuties              https://www.pets.com	200
	`

	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "_redirects"), []byte(data), 0664))

	bundler := runner.NewBundler(s.config)
	artifacts := runner.Artifacts{}

	s.NoError(bundler.ParseRedirects(&artifacts))
	s.Len(artifacts.Redirects, 4)

	s.Equal("/home", artifacts.Redirects[0].From)
	s.Equal("/", artifacts.Redirects[0].To)
	s.Equal(301, artifacts.Redirects[0].Status)

	s.Equal("/blog/my-post.php", artifacts.Redirects[1].From)
	s.Equal("/blog/my-post", artifacts.Redirects[1].To)
	s.Equal(302, artifacts.Redirects[1].Status)

	s.Equal("/news/*", artifacts.Redirects[2].From)
	s.Equal("/blog/$1", artifacts.Redirects[2].To)
	s.Equal(307, artifacts.Redirects[2].Status)

	s.Equal("/cuties", artifacts.Redirects[3].From)
	s.Equal("https://www.pets.com", artifacts.Redirects[3].To)
	s.Equal(0, artifacts.Redirects[3].Status)
}

func (s *BundlerSuite) Test_Zip() {
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".stormkit", "public"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, ".stormkit", "public", ".hidden-file"), []byte("Hello world"), 0664))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".stormkit", "server"), 0774))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".stormkit", "api"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, ".stormkit", "server", "index.js"), []byte("Hello world"), 0664))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "migrations"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "migrations", "01.up.sql"), []byte("create table test ()"), 0664))

	text := "Some data that will be injected into stormkit-api.mjs"
	runner.APIWrapper = text

	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)

	s.NoError(bundler.Zip(artifacts))

	// We should have these files
	for _, zip := range []string{"sk-client.zip", "sk-api.zip", "sk-server.zip", "sk-migrations.zip"} {
		_, err := os.ReadFile(path.Join(s.config.RootDir, "dist", zip))
		s.NoError(err)
	}
}

func (s *BundlerSuite) Test_Zip_MigrationPathDoesNotExist() {
	bundler := runner.NewBundler(s.config)
	artifacts, err := bundler.Bundle(context.Background())

	s.NoError(err)

	err = bundler.Zip(artifacts)

	s.Error(err)
	s.Contains(err.Error(), "migrations")
}

func (s *BundlerSuite) Test_ParseHeaders() {
	headers := `
		# a path:
		/templates/index.html
			# headers for that path:
			X-Frame-Options: DENY
			X-XSS-Protection: 1; mode=block
		# another path:
		/templates/index2.html
			# headers for that path:
			X-Frame-Options: SAMEORIGIN
		/*.jpg
			! Content-Security-Policy
	`

	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "src", "my"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "src", "my", "_headers.txt"), []byte(headers), 0664))

	s.config.Build.HeadersFile = "src/my/_headers.txt"

	artifacts := &runner.Artifacts{}
	bundler := runner.NewBundler(s.config)

	s.NoError(bundler.ParseHeaders(artifacts))

	s.Len(artifacts.Headers, 4)

	s.Equal("X-Frame-Options", artifacts.Headers[0].Key)
	s.Equal("DENY", artifacts.Headers[0].Value)
	s.Equal("/templates/index.html", artifacts.Headers[0].Location)

	s.Equal("X-XSS-Protection", artifacts.Headers[1].Key)
	s.Equal("1; mode=block", artifacts.Headers[1].Value)
	s.Equal("/templates/index.html", artifacts.Headers[1].Location)

	s.Equal("X-Frame-Options", artifacts.Headers[2].Key)
	s.Equal("SAMEORIGIN", artifacts.Headers[2].Value)
	s.Equal("/templates/index2.html", artifacts.Headers[2].Location)

	s.Equal("Content-Security-Policy", artifacts.Headers[3].Key)
	s.Equal("", artifacts.Headers[3].Value)
	s.Equal("/*.jpg", artifacts.Headers[3].Location)
}

func (s *BundlerSuite) Test_CDNFiles() {
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, "dist", "client", "templates"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "dist", "client", "templates", "index.html"), []byte("hello-world"), 0664))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "dist", "client", "templates", "index2.html"), []byte("hello-world-2"), 0664))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "dist", "client", "templates", "my.jpg"), []byte("jpg-content"), 0664))

	var err error
	artifacts := runner.NewArtifacts(s.config.Repo.Dir)
	artifacts.ClientDirs = []string{"/dist/client"}
	artifacts.Headers, err = deploy.ParseHeaders(`
		/templates/index.html
			x-fr: DENY
			x-pr: 1; mode=block
		/templates/index2.html
			x-fr: SAMEORIGIN
		/*.jpg
			! x-sk
	`)

	s.NoError(err)

	cdnFiles := artifacts.CDNFiles()
	s.Equal([]deploy.CDNFile{
		{
			Name: "/templates/index.html",
			Headers: map[string]string{
				"x-fr": "DENY",
				"x-pr": "1; mode=block",
				"etag": `"20-fbb969117edfa916b86dfb67fd11decf1e336df0"`,
			},
		},
		{
			Name: "/templates/index2.html",
			Headers: map[string]string{
				"x-fr": "SAMEORIGIN",
				"etag": `"20-3d0b9a7a7d1882824e95fcc401aedf12d9fc7106"`,
			},
		},
		{
			Name: "/templates/my.jpg",
			Headers: map[string]string{
				"x-sk": "",
				"etag": `"20-9f879f26935916f6950366bc0bbd977effaded83"`,
			},
		},
	}, cdnFiles)
}

func (s *BundlerSuite) Test_RegexpPattern() {
	contents := []string{
		// Invalid import:
		`do not import me`,
		"require(`template-${string}-module`);",
		`
		/**
		 * This is an Express middleware for applications wanting to the define
		 * API routes programmatically.
		 *
		 * Example usage:
		 *
		 * import { apiMiddlewareExpress } from "@stormkit/serverless/middlewares"
		 *
		 * app.use("/api", apiMiddlewareExpress({ apiDir: "src/api", bundler: "vite" }));
		 */
		`,

		// Valid imports:
		`import defaultExport from "first-module";`,
		`import * as name from "ModuleTwo";`,
		`import { export1 } from "third_module";`,
		`import { export1 as alias1 } from "fourth-package";`,
		`import { export1, export2 } from "FifthModule";`,
		`import { export1, export2 as alias2 } from "sixth_package";`,
		`import { "string name" as alias } from "seventh-module";`,
		`import defaultExport, { export1 } from "EighthPackage";`,
		`import defaultExport, * as name from "@ninth/module";`,
		`import "TenthPackage";`,
		`const name = require('eleventh-module');`,
		`const { name } = require('twelfth_package');`,
		`const name = await import('ThirteenthModule');`,
		`const { default: defaultExport } = await import('fourteenth-package');`,
		`import defaultExport, { export1, export2 as alias2, /* ... */ } from "fifteenth_module";`,
		`const { export1, export2: alias2, /* ... */ } = require('SixteenthPackage');`,
		`import { name1, name2 as alias2, /* ... */ } from "seventeenth-module";`,
		`const { name1, name2: alias2, /* ... */ } = require('eighteenth_package');`,
	}

	re := regexp.MustCompile(runner.FindDependencyRegexp)
	matches := []string{}

	for _, c := range contents {
		fileMatches := re.FindAllStringSubmatch(runner.RemoveJSComments(c), -1)

		for _, match := range fileMatches {
			if lm := len(match); lm > 0 {
				matches = append(matches, match[lm-1])
			}
		}
	}

	s.Equal([]string{
		"first-module",
		"ModuleTwo",
		"third_module",
		"fourth-package",
		"FifthModule",
		"sixth_package",
		"seventh-module",
		"EighthPackage",
		"@ninth/module",
		"TenthPackage",
		"eleventh-module",
		"twelfth_package",
		"ThirteenthModule",
		"fourteenth-package",
		"fifteenth_module",
		"SixteenthPackage",
		"seventeenth-module",
		"eighteenth_package",
	}, matches)
}

func Test_BundlerSuite(t *testing.T) {
	suite.Run(t, &BundlerSuite{})
}
