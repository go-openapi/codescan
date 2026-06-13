// SPDX-License-Identifier: Apache-2.0

// Package nullable holds the annotated declarations used by the
// "Nullable pointers" how-to. nullable_test.go scans it twice — with
// SetXNullableForPointers off and on — and writes the before/after golden
// fragments the guide renders.
package nullable

// snippet:model

// Profile has required and optional (pointer) fields.
//
// swagger:model
type Profile struct {
	// Name is always present.
	Name string `json:"name"`

	// Nickname is optional.
	Nickname *string `json:"nickname"`

	// Age is optional.
	Age *int32 `json:"age"`
}

// endsnippet:model
