package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Unix is the unix representation of time.
type Unix struct {
	time.Time
	Valid bool
}

// NewUnix creates a new unix instance.
var NewUnix = func() Unix {
	return Unix{
		Time:  time.Now().UTC(),
		Valid: true,
	}
}

// UnixFrom returns a unix time in UTC.
func UnixFrom(t time.Time) Unix {
	return Unix{
		Time:  t.UTC(),
		Valid: true,
	}
}

// Scan implements the Scanner interface.
func (ut *Unix) Scan(value any) error {
	if value != nil {
		ut.Time, ut.Valid = value.(time.Time)
	}

	return nil
}

// Unix is a wrapper around the unix function.
func (ut *Unix) Unix() int64 {
	if !ut.Valid {
		return 0
	}

	return ut.Time.Unix()
}

// UnixStr returns the unix timestamp (seconds) in a string format.
// This is crucial for Javascript apps because of the integer overflow.
func (ut *Unix) UnixStr() string {
	unix := ut.Unix()

	if unix == 0 {
		return ""
	}

	return strconv.FormatInt(unix, 10)
}

// Value implements the Sql Driver interface.
func (ut Unix) Value() (driver.Value, error) {
	if ut.Valid && !ut.Time.IsZero() {
		return ut.Time.UTC(), nil
	}

	return nil, nil
}

// MarshalJSON implements the Marshaler interface.
func (ut Unix) MarshalJSON() ([]byte, error) {
	if ut.Valid && !ut.Time.IsZero() {
		return json.Marshal(ut.Time.Unix())
	}

	return json.Marshal(nil)
}

// UnmarshalJSON implements the Marshaler interface.
func (ut *Unix) UnmarshalJSON(b []byte) error {
	if b != nil {
		var timestamp int64
		_ = json.Unmarshal(b, &timestamp)
		ut.Time = time.Unix(timestamp, 0)
		ut.Valid = true
	}

	return nil
}

type Map map[string]any

// Value implements the Sql Driver interface.
func (m Map) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the Scanner interface.
func (m *Map) Scan(value any) error {
	// Early return if nil
	if value == nil {
		*m = nil
		return nil
	}

	// Initialize map if nil
	if *m == nil {
		*m = make(map[string]any)
	}

	// Convert to bytes
	var bytes []byte

	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("unsupported type for Map.Scan: %T", value)
	}

	// Handle empty string/bytes
	if len(bytes) == 0 {
		*m = make(map[string]any)
		return nil
	}

	// Unmarshal
	if err := json.Unmarshal(bytes, m); err != nil {
		return fmt.Errorf("failed to unmarshal Map: %w", err)
	}

	return nil
}

func Ptr[T any](v T) *T {
	return &v
}
