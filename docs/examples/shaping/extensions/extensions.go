// SPDX-License-Identifier: Apache-2.0

// Package extensions holds the annotated declarations used by the
// "Vendor extensions" how-to. extensions_test.go scans it with SkipExtensions
// off and on and writes the before/after golden fragments.
package extensions

// snippet:model

// Widget is a small model.
//
// codescan records each field's Go origin as vendor extensions unless
// SkipExtensions is set.
//
// swagger:model
type Widget struct {
	// Label is the display label.
	Label string `json:"label"`

	// Size is the widget size in pixels.
	Size int32 `json:"size"`
}

// endsnippet:model
