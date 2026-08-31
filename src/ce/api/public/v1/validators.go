package publicapiv1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type Validators struct{}

func (v *Validators) ToInt(raw, paramName string) (int, error) {
	if raw == "" {
		return 0, nil
	}

	i, err := strconv.Atoi(raw)

	if err != nil {
		return 0, fmt.Errorf("The '%s' parameter must be a valid integer", paramName)
	}

	if i < 0 {
		return 0, fmt.Errorf("The '%s' parameter cannot be smaller than 0", paramName)
	}

	return i, nil
}

// NormalizeRepo normalizes repo to lower-case and validates that it matches the
// full provider/org/repo shape for a known VCS provider. It returns the
// normalized value and whether the value is valid. An empty string is
// considered valid (no filter applied) and is returned as-is.
func (v *Validators) NormalizeRepo(repo string) (string, bool) {
	if repo == "" {
		return "", true
	}

	// Accept `file://` URLs and stormkit-style references uniformly:
	// derive the canonical "<provider>/<path>" storage form. Only enabled
	// in development — `local` repos are a dev convenience and must not be
	// reachable on Cloud or self-hosted deployments.
	if config.IsDevelopment() {
		if provider, slug := utils.ParseRepoWithProvider(repo); provider != "" && slug != "" {
			if provider == "local" {
				return provider + "/" + slug, true
			}
		}
	}

	normalized := strings.ToLower(repo)
	parts := strings.Split(normalized, "/")

	// Require at least provider + org + repo segments, all non-empty.
	if len(parts) < 3 {
		return "", false
	}

	for _, part := range parts {
		if part == "" {
			return "", false
		}
	}

	validProviders := map[string]struct{}{
		"github":    {},
		"gitlab":    {},
		"bitbucket": {},
	}

	if _, ok := validProviders[parts[0]]; !ok {
		return "", false
	}

	return normalized, true
}
