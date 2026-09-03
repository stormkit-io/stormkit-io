package nixstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

// ProfilesDir holds one Nix profile per environment. It is a variable so tests
// can point it at a temporary directory.
//
// A profile is a garbage collection root. Without one, the only thing keeping
// an environment's dev shell alive is its running process, so an idle service
// loses its packages on the next collection and has to download them again
// before it can answer the request that woke it up.
var ProfilesDir = "/nix/var/nix/profiles/stormkit"

// unsafeProfileChars matches everything that must not reach a file name.
var unsafeProfileChars = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// generationSuffix matches the per-generation links Nix maintains next to a
// profile, for example "app-1-env-2-13-link".
var generationSuffix = regexp.MustCompile(`-[0-9]+-link$`)

// ProfileParams identifies the environment that owns a profile.
type ProfileParams struct {
	AppID string
	EnvID string
}

// ProfilePath returns the profile that roots an environment's dev shell, or an
// empty string when the environment cannot be identified. Callers fall back to
// an unrooted `nix develop` in that case: a missing root costs a re-download,
// a failed deployment costs more.
func ProfilePath(p ProfileParams) string {
	appID := unsafeProfileChars.ReplaceAllString(p.AppID, "")
	envID := unsafeProfileChars.ReplaceAllString(p.EnvID, "")

	if appID == "" || envID == "" {
		return ""
	}

	return filepath.Join(ProfilesDir, fmt.Sprintf("app-%s-env-%s", appID, envID))
}

// EnsureProfilesDir creates the directory that holds environment profiles.
func EnsureProfilesDir() error {
	return os.MkdirAll(ProfilesDir, 0o755)
}

// Profile is a registered environment root and the time it was last deployed.
type Profile struct {
	Name     string
	Path     string
	Modified time.Time
}

// ListProfiles returns the environment profiles, most recently deployed first.
// A missing directory is not an error: nothing has been deployed yet.
func ListProfiles() ([]Profile, error) {
	entries, err := os.ReadDir(ProfilesDir)

	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	profiles := make([]Profile, 0, len(entries))

	for _, entry := range entries {
		name := entry.Name()

		// Generation links belong to a profile and are removed along with it.
		if generationSuffix.MatchString(name) {
			continue
		}

		info, err := entry.Info()

		if err != nil {
			continue
		}

		profiles = append(profiles, Profile{
			Name:     name,
			Path:     filepath.Join(ProfilesDir, name),
			Modified: info.ModTime(),
		})
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Modified.After(profiles[j].Modified)
	})

	return profiles, nil
}

// RemoveProfile deletes a profile and every generation link belonging to it,
// which makes its closure collectable on the next run.
func RemoveProfile(p Profile) error {
	links, err := filepath.Glob(p.Path + "-*-link")

	if err != nil {
		return err
	}

	for _, link := range append(links, p.Path) {
		if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// ReapProfilesParams configures which environment roots are dropped.
type ReapProfilesParams struct {
	// RetentionDays drops profiles that have not been deployed within this
	// many days. Values below 1 fall back to DefaultRetentionDays.
	RetentionDays int
}

// ReapProfiles removes the roots of environments that have not deployed within
// the retention window, so the next collection can reclaim their packages.
func ReapProfiles(p ReapProfilesParams) (int, error) {
	if p.RetentionDays < 1 {
		p.RetentionDays = DefaultRetentionDays
	}

	profiles, err := ListProfiles()

	if err != nil {
		return 0, err
	}

	cutoff := time.Now().AddDate(0, 0, -p.RetentionDays)
	removed := 0

	for _, profile := range profiles {
		if profile.Modified.After(cutoff) {
			continue
		}

		if err := RemoveProfile(profile); err != nil {
			return removed, err
		}

		removed++
	}

	return removed, nil
}

// PruneGenerationsParams configures generation pruning.
type PruneGenerationsParams struct {
	ProfilePath string
}

// PruneGenerations deletes every generation of a profile except the newest.
//
// Nix keeps the previous version on every write, and each generation is a root
// of its own. Without pruning, an environment that deploys daily pins a new
// toolchain every day and the store grows per deployment rather than per
// environment.
func PruneGenerations(ctx context.Context, p PruneGenerationsParams) error {
	if p.ProfilePath == "" {
		return nil
	}

	out, err := sys.Command(ctx, sys.CommandOpts{
		Name: "nix-env",
		Args: []string{"--profile", p.ProfilePath, "--delete-generations", "+1"},
	}).CombinedOutput()

	if err != nil {
		return fmt.Errorf("nix-env --delete-generations %s: %w: %s", p.ProfilePath, err, strings.TrimSpace(string(out)))
	}

	return nil
}

// PruneAllGenerations prunes every registered environment profile, returning
// the first error while still pruning the rest.
func PruneAllGenerations(ctx context.Context) error {
	profiles, err := ListProfiles()

	if err != nil {
		return err
	}

	var firstErr error

	for _, profile := range profiles {
		if err := PruneGenerations(ctx, PruneGenerationsParams{ProfilePath: profile.Path}); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
