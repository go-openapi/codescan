// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package modeloverridesimpleschema is the F1/F4 simple-schema guard. A
// model-annotated type carrying an override (enum / strfmt) is referenced by
// $ref in full-schema mode, but $ref is illegal in an OAS-2 SimpleSchema
// (non-body parameters, response headers). There the override must still
// INLINE. This pins that the model-vs-$ref gate does not leak a $ref into a
// simple schema.
package modeloverridesimpleschema

// Color is an enum that is also swagger:model.
//
// swagger:enum Color
// swagger:model Color
type Color string

// Color values.
const (
	// ColorRed is red.
	ColorRed Color = "red"
	// ColorBlue is blue.
	ColorBlue Color = "blue"
)

// SKU is a string-format type that is also swagger:model.
//
// swagger:strfmt sku
// swagger:model SKU
type SKU string

// listThingsParams are query parameters referencing the model override types.
//
// swagger:parameters listThings
type listThingsParams struct {
	// Filter by color (enum), in a non-body parameter.
	//
	// in: query
	Filter Color `json:"filter"`
}

// thingResponse carries a response header referencing a model override type.
//
// swagger:response thingResponse
type thingResponse struct {
	// Sku is a response header (strfmt).
	//
	// in: header
	Sku SKU `json:"X-Sku"`
}

// swagger:route GET /things listThings
//
// responses:
//
//	200: thingResponse
func listThings() {}
