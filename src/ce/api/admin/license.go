package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const LicenseVersion20240610 = "2024-06-10"
const LicenseVersion20241210 = "2024-12-10"
const LicenseVersion20250926 = "2025-09-26"
const MaximumFreeSeats = -1 // -1 means unlimited

var CachedLicense *License
var mux sync.Mutex

type License struct {
	Key      string         `json:"key"`
	Version  string         `json:"version"`
	Seats    int            `json:"seats"`
	UserID   types.ID       `json:"userId,omitempty"`
	Premium  bool           `json:"premium"`  // Enables premium features
	Ultimate bool           `json:"ultimate"` // Enables ultimate features
	Metadata map[string]any `json:"metadata,omitempty"`
}

type NewLicenseArgs struct {
	Seats    int
	Key      string
	UserID   types.ID
	Premium  bool
	Ultimate bool
	Metadata map[string]any
}

func NewLicense(args NewLicenseArgs) *License {
	key := args.Key

	if key == "" {
		key = utils.RandomToken(128)
	}

	return &License{
		Key:      key,
		Seats:    args.Seats,
		UserID:   args.UserID,
		Version:  LicenseVersion20250926,
		Premium:  args.Premium,
		Ultimate: args.Ultimate,
		Metadata: args.Metadata,
	}
}

// CurrentLicense returns the current license, loading it if necessary.
func CurrentLicense() *License {
	mux.Lock()
	defer mux.Unlock()

	if config.IsStormkitCloud() {
		CachedLicense = &License{
			Seats:   -1,
			Key:     "stormkit-cloud",
			Version: LicenseVersion20250926,
			Premium: true,
		}

		return CachedLicense
	}

	if CachedLicense == nil {
		cnf, err := Store().Config(context.Background())

		if err != nil {
			slog.Errorf("error while retrieving admin config from db: %v", err)
		}

		var key string

		if cnf.LicenseConfig != nil && cnf.LicenseConfig.Key != "" {
			key = cnf.LicenseConfig.Key
		} else if key = os.Getenv("STORMKIT_LICENSE"); key != "" {
			// Backwards compatibility: we also support loading the license from an env var
			cnf.LicenseConfig = &LicenseConfig{
				Key: key,
			}
		}

		if key == "" {
			CachedLicense = FreeLicense()
			return CachedLicense
		}

		license, err := ValidateLicense(key)

		if err != nil {
			slog.Errorf("error while validating license: %v", err)
		}

		if license == nil {
			license = FreeLicense()
		}

		CachedLicense = license
	}

	return CachedLicense
}

// ResetLicense sets the current license in memory.
func ResetLicense() {
	slog.Debug(slog.LogOpts{
		Msg:   "invalidating license cache",
		Level: slog.DL1,
	})

	mux.Lock()
	CachedLicense = nil
	mux.Unlock()

	CurrentLicense()
}

type LicenseResponse struct {
	License License `json:"license"`
}

func FreeLicense() *License {
	return NewLicense(NewLicenseArgs{
		Seats: MaximumFreeSeats,
	})
}

// ValidateLicense checks if the provided license key is valid by calling the Stormkit API.
func ValidateLicense(key string) (*License, error) {
	url := fmt.Sprintf("https://api.stormkit.io/v1/license/check?token=%s", key)
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	response, err := shttp.
		NewRequestV2(shttp.MethodGet, url).
		Headers(headers).
		WithExponentialBackoff(5*time.Minute, 10).
		Do()

	if err != nil || response == nil {
		return nil, fmt.Errorf("error while checking license: %v", err)
	}

	if response.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("license is either invalid or no longer active")
	}

	body, err := io.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	data := LicenseResponse{
		License: License{},
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if data.License.Seats < 1 {
		return nil, fmt.Errorf("license is either invalid or no longer active")
	}

	totalUsers, err := Store().TotalUsers(context.Background())

	if err != nil {
		return nil, fmt.Errorf("error while counting users: %v", err)
	}

	if int64(data.License.Seats) < totalUsers {
		return nil, fmt.Errorf("license only allows %d seats, but there are %d users in the system", data.License.Seats, totalUsers)
	}

	// We need to generate a full-blown license because we're going to store it in the DB
	return &License{
		Seats:    data.License.Seats,
		Version:  data.License.Version,
		Premium:  data.License.Premium,
		Ultimate: data.License.Ultimate,
		Key:      key,
	}, nil
}

// Token encrypts the claims and creates an encrypted string from it.
func (l *License) Token() string {
	return fmt.Sprintf("%s:%s", l.UserID.String(), l.Key)
}

// IsEmpty returns true if the license is empty (i.e. has no key and zero seats).
func (l *License) IsEmpty() bool {
	return l.Key == "" && l.Seats == 0
}

// IsEnterprise returns true if the license is an enterprise license.
func (l *License) IsEnterprise() bool {
	return l.Premium || l.Ultimate
}

// Edition returns the edition of the license: "community" or "enterprise".
func (l *License) Edition() string {
	if l.IsEnterprise() {
		return "enterprise"
	}

	return "community"
}

// Debug prints the license information to the stdout
func (l *License) Debug() {
	seats := fmt.Sprintf("%d", l.Seats)

	if l.Seats == -1 {
		seats = "unlimited"
	}

	slog.Infof("%s license active with %s seats", l.Edition(), seats)
}

// SetMockLicense returns a mock license for testing purposes only.
func SetMockLicense() {
	if !config.IsTest() {
		panic("MockLicense can only be used in test environments")
	}

	// If we're setting a mock license, it means we're in a self-hosted environment
	config.SetIsSelfHosted(true)

	mux.Lock()
	defer mux.Unlock()

	CachedLicense = &License{
		Seats:    10,
		Key:      "abcd-efgh-1234-defg-5829-bnac-00",
		Version:  LicenseVersion20250926,
		Premium:  true,
		Ultimate: false,
	}
}

// ResetMockLicense resets the mock license.
func ResetMockLicense() {
	if !config.IsTest() {
		panic("MockLicense can only be used in test environments")
	}

	mux.Lock()
	CachedLicense = nil
	mux.Unlock()
}
