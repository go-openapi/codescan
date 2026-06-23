// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package response_examples_by_mime witnesses response-level `examples`
// (a map keyed by mime type) on a struct-based `swagger:response`
// (go-swagger#2871).
//
// OAS2 reserves the plural `examples` keyword for the Response object;
// the value is a `{mimeType: example}` map. The `swagger:operation` YAML
// path already carries this for free (it unmarshals straight into the
// spec types — see fixtures/bugs/1713 and fixtures/bugs/2871); this
// fixture exercises the Go-struct `swagger:response` path, where the
// `examples:` block in the decl comment is parsed into Response.examples.
package response_examples_by_mime

// Widget is the response payload.
//
// swagger:model Widget
type Widget struct {
	Name string `json:"name"`
}

// WidgetResponse returns a widget, with response-level examples per mime type.
//
// swagger:response widgetResponse
//
// examples:
//
//	application/json:
//	  name: alice
//	  count: 3
//	application/xml: "<widget><name>alice</name></widget>"
type WidgetResponse struct {
	// in: body
	Body Widget `json:"body"`
}

// GetWidgets swagger:route GET /widgets widgets getWidgets
//
// Get a widget.
//
// Responses:
//
//	200: widgetResponse
func GetWidgets() {}
