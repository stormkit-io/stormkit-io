// Package nixstoretest provides helpers for tests that exercise code paths
// guarded by nixstore.Available.
package nixstoretest

import (
	"errors"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
)

// StubLookPath makes nixstore.Available behave as if the Nix binaries are (or
// are not) installed, and returns a function that restores the real lookup.
func StubLookPath(found bool) (restore func()) {
	original := nixstore.LookPath

	if found {
		nixstore.LookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	} else {
		nixstore.LookPath = func(string) (string, error) { return "", errors.New("not found") }
	}

	return func() { nixstore.LookPath = original }
}
