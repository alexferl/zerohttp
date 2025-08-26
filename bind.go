package zerohttp

import (
	"encoding/json"
	"io"
)

// Bind is the default binder instance used by the package
var Bind Binder = &defaultBinder{}

// B is a short alias for Bind for convenience
var B = Bind

// Binder handles request binding and parsing for various content types.
// It provides methods to decode request data into Go structs.
type Binder interface {
	// JSON decodes JSON request body into the destination struct.
	// It uses json.NewDecoder with DisallowUnknownFields enabled
	// for safer JSON parsing that rejects unknown fields.
	JSON(r io.Reader, dst any) error
}

// defaultBinder implements the Binder interface with standard JSON decoding
type defaultBinder struct{}

// JSON decodes JSON request body into the destination struct.
// It configures the decoder to disallow unknown fields for stricter validation.
// Returns an error if the JSON is malformed or contains unknown fields.
func (b *defaultBinder) JSON(r io.Reader, dst any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
