// SPDX-License-Identifier: Apache-2.0

// Package overlay holds the annotated declarations used by the "Overlaying a
// spec" how-to. overlay_test.go scans it on top of a hand-authored base spec
// (Options.InputSpec) and writes the before/after golden fragments.
package overlay

// snippet:model

// Widget is discovered by the scan and merged onto the input spec.
//
// swagger:model
type Widget struct {
	// ID identifies the widget.
	ID string `json:"id"`
}

// endsnippet:model
