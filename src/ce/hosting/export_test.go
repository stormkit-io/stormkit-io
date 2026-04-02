package hosting

import "github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"

// SetFetchConfigFn replaces the function used to fetch app config from the
// database. It is intended for use in tests only.
func SetFetchConfigFn(fn func(string) ([]*appconf.Config, error)) {
	fetchConfigFn = fn
}

// ResetFetchConfigFn restores the default database-backed fetch function.
func ResetFetchConfigFn() {
	fetchConfigFn = appconf.FetchConfig
}

// InvalidateAppCache removes a hostname entry from the in-process cache,
// allowing tests to force a cold-cache scenario without restarting the process.
func InvalidateAppCache(hostName string) {
	appCacheMu.Lock()
	delete(appCache, hostName)
	appCacheMu.Unlock()
}
