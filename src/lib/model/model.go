package model

import "github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"

// Model is an interface for models that need validation.
// Models implementing this interface will be validated upon
// non-get requests.
type Model interface {

	// Validate is the validation function that will be called on
	// the model. If there are no errors, return nil.
	Validate() *shttperr.ValidationError
}
