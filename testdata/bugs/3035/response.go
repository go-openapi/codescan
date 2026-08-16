// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug3035 reproduces go-swagger issue #3035 ("Example spec for
// swagger:response does not produce example output"): a swagger:response
// whose body is an anonymous inline struct used to emit only the response
// description, with no schema at all — so field-level Example/Required and
// property descriptions were lost.
//
// The expected behaviour — locked by the golden — is that the inline body
// struct produces a full object schema carrying its required set, property
// descriptions, and per-field examples.
//
// Note: this fixture differs from the issue's original snippet by a single
// blank line after the body field's leading prose ("The error message"),
// which is the canonical annotation form. The field-level descriptions and
// example are captured regardless; the body field's leading prose is not
// propagated to the schema description (codescan does not surface it — see
// TestCoverage_Bug3035).
package bug3035

// A ValidationError is an error that is used when the required input fails validation.
// swagger:response validationError
type ValidationError struct {
	// The error message
	//
	// in: body
	Body struct {
		// The validation message
		//
		// Required: true
		// Example: Expected type int
		Message string
		// An optional field name to which this validation applies
		FieldName string
	}
}
