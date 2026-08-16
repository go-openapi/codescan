// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package objectkeywords exercises the context gating of the object
// validation keywords (minProperties / maxProperties / patternProperties):
//
//   - on an object-typed model they are kept (the happy path);
//   - on a non-object (scalar) model they are stripped with a
//     CodeShapeMismatch diagnostic — the top-level decl re-check, since
//     the model's Go type is only resolved after the doc block is
//     dispatched;
//   - on a SimpleSchema (query) parameter they are dropped with a
//     CodeUnsupportedInSimpleSchema diagnostic — object validations have
//     no SimpleSchema form in OAS v2.
package objectkeywords

// ObjectModel is a free-form object that legitimately carries the object
// validation keywords — they are kept on the schema.
//
// minProperties: 1
// maxProperties: 9
// patternProperties: ^x-
//
// swagger:model ObjectModel
type ObjectModel map[string]any

// ScalarModel is a string model that wrongly carries object validation
// keywords. They are stripped (with a shape-mismatch diagnostic) because
// the resolved type is string, not object.
//
// minProperties: 1
// maxProperties: 9
// patternProperties: ^x-
//
// swagger:model ScalarModel
type ScalarModel string

// QueryParams carries object keywords on a SimpleSchema (query) parameter;
// they are dropped with an unsupported-in-simple-schema diagnostic.
//
// swagger:parameters listThings
type QueryParams struct {
	// Filter is a simple query parameter.
	//
	// in: query
	// minProperties: 1
	// maxProperties: 9
	// patternProperties: ^x-
	Filter string `json:"filter"`
}

// ListThings lists things.
//
// swagger:route GET /things listThings
//
// Responses:
//   200: body:string
func ListThings() {}
