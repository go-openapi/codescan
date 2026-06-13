// SPDX-License-Identifier: Apache-2.0

// Package discovery holds the annotated declarations used by the "When the
// scanner emits a type" how-to. discovery_test.go scans it and writes the
// golden fragment the guide renders.
package discovery

// snippet:types

// Order is reached only through Cart below. A referenced named type is emitted
// as a $ref target even without swagger:model.
type Order struct {
	// ID is the order identifier.
	ID string `json:"id"`
}

// Cart references Order, so Order gets a definition and the field a $ref.
//
// swagger:model
type Cart struct {
	// Order is the referenced (and therefore emitted) nested model.
	Order Order `json:"order"`
}

// Standalone is never referenced, but swagger:model together with ScanModels
// publishes it anyway.
//
// swagger:model
type Standalone struct {
	// Label is a free-text label.
	Label string `json:"label"`
}

// Orphan is never referenced and carries no swagger:model — the scanner does
// not invent it, so it never reaches the spec.
type Orphan struct {
	// Secret is internal.
	Secret string `json:"secret"`
}

// endsnippet:types
