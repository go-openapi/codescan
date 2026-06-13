// SPDX-License-Identifier: Apache-2.0

// Package descref holds the annotated declarations used by the "Descriptions
// beside a $ref" how-to. descref_test.go scans it with DescWithRef off and on
// and writes the before/after golden fragments.
package descref

// snippet:model

// Address is a referenced model.
//
// swagger:model
type Address struct {
	// Street is the street line.
	Street string `json:"street"`
}

// Person references Address through a field whose only decoration is a
// description.
//
// swagger:model
type Person struct {
	// Home is where the person lives.
	Home Address `json:"home"`
}

// endsnippet:model
