// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug3069 demonstrates the resolution to go-swagger issue #3069 ("Is
// there a way to change the representation of one parameter of the request
// object?"). ComplexType has a custom JSON marshaller that emits a dotted
// string, so its struct shape (A/B/C) does not match the wire format. codescan
// cannot infer a custom MarshalJSON, but a `swagger:type` override (or
// implementing encoding.TextMarshaler / swagger:strfmt) lets the author state
// the representation explicitly — here, string — so a []ComplexType field
// renders as an array of strings.
package bug3069

// ComplexType marshals to a dotted string; the override states that.
//
// swagger:type string
type ComplexType struct {
	A string
	B string
	C string
}

// swagger:model MyResponse
type MyResponse struct {
	// List of complex types.
	ComplexType []ComplexType `json:"complexType"`
}
