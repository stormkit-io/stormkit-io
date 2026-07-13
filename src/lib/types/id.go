package types

import (
	"encoding/json"
	"fmt"
)

type ID int64

func (id ID) String() string {
	return fmt.Sprintf("%d", id)
}

// MarshalJSON implements the json.Marshaler interface.
func (id ID) MarshalJSON() ([]byte, error) {
	if id == 0 {
		return json.Marshal("")
	}

	return json.Marshal(fmt.Sprintf("%d", id))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It unmarshals a JSON string back to the ID type.
func (id *ID) UnmarshalJSON(data []byte) error {
	var s string
	var intValue int64

	if err := json.Unmarshal(data, &s); err != nil {
		// Try the int value if string fails
		if err := json.Unmarshal(data, &intValue); err != nil {
			return err
		}
	}

	if s != "" {
		if _, err := fmt.Sscan(s, &intValue); err != nil {
			return err
		}
	}

	*id = ID(intValue)
	return nil
}
