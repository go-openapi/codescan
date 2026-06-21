// SPDX-License-Identifier: Apache-2.0

// Package refsiblings holds the annotated declarations used by the "$ref
// siblings" section of the "Descriptions beside a $ref" how-to.
// refsiblings_test.go scans it under the default, EmitRefSiblings and
// SkipAllOfCompounding options and writes the golden fragments the page renders.
package refsiblings

// snippet:model

// Address is a referenced model.
//
// swagger:model
type Address struct {
	// Street is the street line.
	Street string `json:"street"`
}

// Person references Address through a field decorated with a description and a
// vendor extension — both can, in principle, sit beside the $ref. How they are
// rendered depends on the options.
//
// swagger:model
type Person struct {
	// Home is where the person lives.
	//
	// extensions:
	//   x-ui-order: 3
	Home Address `json:"home"`
}

// endsnippet:model
