package nixstore

import "errors"

// SetLookPath lets tests simulate a container with or without the Nix binaries.
func SetLookPath(found bool) (restore func()) {
	original := lookPath

	if found {
		lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	} else {
		lookPath = func(string) (string, error) { return "", errors.New("not found") }
	}

	return func() { lookPath = original }
}
