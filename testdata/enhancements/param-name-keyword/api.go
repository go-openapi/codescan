// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package param_name_keyword exercises the `name:` keyword on a
// swagger:parameters struct field. `name:` renames the JSON parameter
// name, overriding the json-tag / Go-field derivation (the parameter-side
// analogue of swagger:name on a schema field). Being a structural keyword
// it is also stripped from the parameter description rather than leaking
// into it as prose.
package param_name_keyword

// swagger:model Widget
type Widget struct {
	Color string `json:"color"`
}

// swagger:parameters createWidget
type createWidgetParams struct {
	// the widget payload
	// in: body
	// name: widget
	Body Widget

	// how many to make
	// in: query
	// name: count
	// minimum: 1
	Quantity int64
}

// swagger:route POST /widgets widgets createWidget
//
// Create a widget.
//
// responses:
//
//	200: description: ok
func createWidget() {}
