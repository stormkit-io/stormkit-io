package shttperr

import (
	"encoding/json"
	"errors"
)

// Error is a custom error
type Error struct {
	error

	// Status is the response status.
	status int

	// Code is the error code.
	code string

	OriginalError error
}

// New creates a new error instance.
func New(status int, msg, code string) *Error {
	return &Error{
		error:  errors.New(msg),
		status: status,
		code:   code,
	}
}

// Status returns the status code.
func (e Error) Status() int {
	return e.status
}

// Code returns the error code.
func (e Error) Code() string {
	return e.code
}

// SetOriginal sets the original error.
func (e *Error) SetOriginal(err error) *Error {
	e.OriginalError = err
	return e
}

// ValidationError is a custom error map which holds validation errors.
type ValidationError struct {
	Errors map[string]string
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	if ve.Errors == nil {
		return ""
	}

	b, _ := json.Marshal(ve.Errors)
	return string(b)
}

// SetError sets an error. If the Errors instance is nil yet,
// it will be created automatically.
func (ve *ValidationError) SetError(key, value string) {
	if ve.Errors == nil {
		ve.Errors = map[string]string{}
	}

	ve.Errors[key] = value
}

// ToError returns nil if the errors map is empty, or returns
// the validation error instance in the other case.
func (ve *ValidationError) ToError() *ValidationError {
	if ve.Errors == nil {
		return nil
	}

	return ve
}
