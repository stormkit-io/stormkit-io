package runner_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stretchr/testify/suite"
)

type DepTree struct {
	suite.Suite
	nodeModules string
}

func (s *DepTree) SetupTest() {
	s.nodeModules = filepath.Join(s.T().TempDir(), "node_modules")

	packages := map[string]string{
		"next":         `{ "dependencies": { "@next/env": "1", "styled-jsx": "1" }, "peerDependencies": { "react": "1" } }`,
		"@next/env":    `{}`,
		"styled-jsx":   `{ "dependencies": { "client-only": "1" }, "peerDependencies": { "react": "1" } }`,
		"client-only":  `{}`,
		"react":        `{ "dependencies": { "loose-envify": "1" } }`,
		"loose-envify": `{ "dependencies": { "react": "1" } }`,
		"unrelated":    `{ "dependencies": { "left-pad": "1" } }`,
	}

	for name, content := range packages {
		dir := filepath.Join(s.nodeModules, name)
		s.Require().NoError(os.MkdirAll(dir, 0755))
		s.Require().NoError(os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644))
	}
}

// Test_Walk verifies that dependencies and peer dependencies are resolved
// transitively, cycles terminate, and packages missing from node_modules are
// still listed.
func (s *DepTree) Test_Walk() {
	tree := runner.NewDepedencyTree([]string{"next", "missing-pkg"}, s.nodeModules)
	tree.Walk()

	names := []string{}

	for _, r := range tree.ResolvedDepedencies() {
		names = append(names, r.Name)
		s.Equal(filepath.Join(s.nodeModules, r.Name), r.FullPath)
	}

	sort.Strings(names)

	s.Equal([]string{
		"@next/env", "client-only", "loose-envify", "missing-pkg",
		"next", "react", "styled-jsx",
	}, names)
}

func TestDependencyTree(t *testing.T) {
	suite.Run(t, &DepTree{})
}
