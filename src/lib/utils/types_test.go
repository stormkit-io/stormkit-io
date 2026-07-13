package utils

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestUnix(t *testing.T) {
	my := struct {
		Time Unix `json:"time"`
	}{}

	if _, err := json.Marshal(my); err != nil {
		t.Fatalf("Was not expecting this error: %s", err.Error())
	}

	my.Time = Unix{time.Now(), true}
	data, err := json.Marshal(my)

	if string(data) != fmt.Sprintf(`{"time":%d}`, my.Time.Unix()) {
		t.Fatalf("Wrong marshaling")
	}

	if err != nil {
		t.Fatalf("Was not expecting this error: %s", err.Error())
	}

	var my2 struct {
		Time Unix `json:"time"`
	}

	if err := json.Unmarshal(data, &my2); err != nil {
		t.Fatalf("Was not expecting this error: %s", err.Error())
	}

	if my2.Time.Unix() != my.Time.Unix() {
		t.Fatalf("Unix timestamps do not match")
	}
}
