package runner_test

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stretchr/testify/suite"
)

type DepTree struct {
	suite.Suite
	config      runner.RunnerOpts
	cacheFolder string
}

func (s *DepTree) SetupTest() {
	home, err := os.UserHomeDir()

	s.NoError(err)
	s.NotEmpty(home)

	runner.DefaultRepo = nil
	runner.DefaultInstaller = nil
	s.cacheFolder = path.Join(home, ".cache", "next-hello-world")

	s.config = runner.RunnerOpts{
		RootDir:  s.cacheFolder,
		WorkDir:  path.Join(s.cacheFolder, "repo"),
		Reporter: runner.NewReporter("http://example.com"),
		Build: runner.BuildOpts{
			BuildCmd:     "npm run build",
			ServerFolder: ".next",
			EnvVars: map[string]string{
				"CI": "true",
			},
		},
		Repo: runner.RepoOpts{
			PackageJson: &runner.PackageJson{},
			Address:     "https://github.com/svedova/next-hello-world",
			Branch:      "main",
			Dir:         path.Join(s.cacheFolder, "repo"),
		},
	}

	ctx := context.Background()

	// Check if the repository exists under the .cache subfolder
	if _, err := os.Stat(s.cacheFolder); os.IsNotExist(err) {
		s.NoError(s.config.MkdirAll())

		// Checkout the repository
		s.NoError(runner.NewRepo(s.config).Checkout(ctx))

		// Install dependencies
		installer := runner.NewInstaller(s.config)
		s.NoError(installer.Install(ctx))

		// Build the app
		builder := runner.NewBuilder(s.config)
		s.NoError(builder.ExecCommands(ctx))
	}
}

// Note: this test has no teardown because we want to keep the cache folder
// between test runs to speed up the process.
func (s *DepTree) Test_Walk() {
	tree := runner.NewDepedencyTree([]string{"next"}, path.Join(s.cacheFolder, "repo", "node_modules"))
	tree.Walk()

	resolved := tree.ResolvedDepedencies()
	names := []string{}

	for _, r := range resolved {
		names = append(names, r.Name)
	}

	sort.Strings(names)

	js, err := json.Marshal(names)
	s.NoError(err)

	s.JSONEq(`[
		"@next/env", "@opentelemetry/api", "@playwright/test",
		"@swc/counter", "@swc/helpers", "busboy", "caniuse-lite",
		"client-only", "graceful-fs", "js-tokens", "loose-envify",
		"nanoid", "next", "picocolors", "postcss", "react", "react-dom",
		"sass", "scheduler", "source-map-js", "streamsearch", "styled-jsx", "tslib"
	]`, string(js))
}

func TestDependencyTree(t *testing.T) {
	suite.Run(t, &DepTree{})
}
