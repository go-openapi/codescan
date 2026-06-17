// SPDX-License-Identifier: Apache-2.0

// Package formats holds the annotated declarations for the "Forcing a
// conformant format" how-to. formats_test.go scans it and writes the golden
// definition the guide renders.
package formats

// snippet:formats

// Measurement forces a JSON-conformant format on a field. A uint64 field emits
// the Go-specific `{integer, format: uint64}` by default; overriding it with a
// field-level `swagger:strfmt int64` (below) yields a precision-safe,
// string-encoded `{string, format: int64}`.
//
// swagger:model
type Measurement struct {
	// Raw keeps the default Go-derived vendor format (uint64).
	Raw uint64 `json:"raw"`

	// Bounded is forced to a conformant, string-encoded int64.
	//
	// swagger:strfmt int64
	Bounded uint64 `json:"bounded"`
}

// endsnippet:formats
