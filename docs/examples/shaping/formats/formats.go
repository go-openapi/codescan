// SPDX-License-Identifier: Apache-2.0

// Package formats holds the annotated declarations for the "Forcing a
// conformant format" how-to. formats_test.go scans it and writes the golden
// definition the guide renders.
package formats

// snippet:formats

// Measurement shows forcing a JSON-conformant format. codescan emits a
// Go-specific {integer, format: uint64} for an unsized/large int by default;
// swagger:strfmt int64 on a field overrides that to a precision-safe,
// JSON-conformant string encoding ({string, format: int64}).
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
