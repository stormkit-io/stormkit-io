package runner

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

type DependencyTree struct {
	pathToNodeModules    string
	requestedDepedencies []string
	resolvedDependencies map[string]bool
}

func NewDepedencyTree(dependencies []string, pathToNodeModules string) *DependencyTree {
	return &DependencyTree{
		resolvedDependencies: map[string]bool{},
		requestedDepedencies: dependencies,
		pathToNodeModules:    pathToNodeModules,
	}
}

func (dt *DependencyTree) Walk(deps ...[]string) {
	var normalized []string

	if len(deps) > 0 {
		normalized = deps[0]
	} else {
		normalized = dt.requestedDepedencies
	}

	for _, dep := range normalized {
		if dt.resolvedDependencies[dep] {
			continue
		}

		dt.resolvedDependencies[dep] = true

		if childDeps := dt.resolveChildDeps(dep); len(childDeps) > 0 {
			dt.Walk(childDeps)
			continue
		}
	}
}

type Depedency struct {
	Name     string
	FullPath string
}

func (dt *DependencyTree) ResolvedDepedencies() []Depedency {
	resolved := []Depedency{}

	for dep := range dt.resolvedDependencies {
		resolved = append(resolved, Depedency{
			Name:     dep,
			FullPath: filepath.Join(dt.pathToNodeModules, dep),
		})
	}

	return resolved
}

func (dt *DependencyTree) resolveChildDeps(dependency string) []string {
	pckJsonPath := filepath.Join(dt.pathToNodeModules, dependency, "package.json")

	// No package json file found - return early
	if _, err := os.Stat(pckJsonPath); err != nil {
		return nil
	}

	file, _ := os.ReadFile(pckJsonPath)
	pj := &PackageJson{}

	if err := json.Unmarshal(file, pj); err != nil {
		slog.Infof("cannot unmarshal package.json: %s", err.Error())
	}

	childDeps := []string{}

	for k := range pj.Dependencies {
		childDeps = append(childDeps, k)
	}

	for k := range pj.PeerDependencies {
		childDeps = append(childDeps, k)
	}

	return childDeps
}
