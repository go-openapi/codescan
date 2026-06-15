// SPDX-License-Identifier: Apache-2.0

// Package refoverride holds the annotated declarations used by the "Decorating a
// $ref" subsection of the "Model definitions" tutorial. refoverride_test.go
// scans it and writes the golden fragment the tutorial renders.
package refoverride

// snippet:refoverride

// Address is a referenced model.
//
// swagger:model
type Address struct {
	// Street is the street line.
	Street string `json:"street"`
}

// Person references Address through a field that also carries a description and
// a vendor extension. A bare $ref cannot hold sibling keywords, so codescan
// wraps the reference as a member of an allOf — the property is then a normal
// schema that keeps the description and the x-* extension, and required goes to
// the parent. Nothing is dropped.
//
// swagger:model
type Person struct {
	// Home is where the person lives.
	//
	// required: true
	// extensions:
	//   x-ui-order: 3
	Home Address `json:"home"`
}

// endsnippet:refoverride
