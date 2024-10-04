package types

import "encoding/json"

// Error Instead of utilizing HTTP status codes to describe node errors (which often do not have a
// good analog), rich errors are returned using this object. Both the code and message fields can be
// individually used to correctly identify an error. Implementations MUST use unique values for both
// fields.
type Error struct {
	// Code is a network-specific error code. If desired, this code can be equivalent to an HTTP
	// status code.
	Code int32 `json:"code"`
	// Message is a network-specific error message. The message MUST NOT change for a given code. In
	// particular, this means that any contextual information should be included in the details
	// field.
	Message string `json:"message"`
	// Description allows the implementer to optionally provide additional information about an
	// error. In many cases, the content of this field will be a copy-and-paste from existing
	// developer documentation. Description can ONLY be populated with generic information about a
	// particular type of error. It MUST NOT be populated with information about a particular
	// instantiation of an error (use `details` for this). Whereas the content of Error.Message
	// should stay stable across releases, the content of Error.Description will likely change
	// across releases (as implementers improve error documentation). For this reason, the content
	// in this field is not part of any type assertion (unlike Error.Message).
	Description *string `json:"description,omitempty"`
	// An error is retriable if the same request may succeed if submitted again.
	Retriable bool `json:"retriable"`
	// Often times it is useful to return context specific to the request that caused the error
	// (i.e. a sample of the stack trace or impacted account) in addition to the standard error
	// message.
	Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string {
	bytes, _ := json.MarshalIndent(e, "", "  ")
	return string(bytes)
}

var (
	ErrInvalidAddress = &Error{
		Code:    12, //nolint
		Message: "Invalid address",
	}
)

// wrapErr adds details to the types.Error provided. We use a function
// to do this so that we don't accidentially overrwrite the standard
// errors.
func WrapErr(rErr *Error, err error) *Error {
	newErr := &Error{
		Code:      rErr.Code,
		Message:   rErr.Message,
		Retriable: rErr.Retriable,
	}
	if err != nil {
		newErr.Details = map[string]interface{}{
			"context": err.Error(),
		}
	}

	return newErr
}
