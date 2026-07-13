package apploghandlers

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func TestServices(t *testing.T) {
	r := shttp.NewRouter()
	s := r.RegisterService(Services)

	if s == nil {
		t.Fatalf("Was expecting service not to be nil")
	}

	handlers := []string{
		"GET:/app/{did:[0-9]+}/logs",
	}

	if reflect.DeepEqual(s.Handlers(), handlers) == false {
		fmt.Println(s.Handlers())
		fmt.Println(handlers)
		t.Fatalf("Handlers are not registered correctly")
	}
}
