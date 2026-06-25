// SPDX-License-Identifier: Apache-2.0

// Package afterdecl is the test-backed witness for the "Keeping annotations
// out of the godoc" how-to. It shows the AfterDeclComments opt-in: swagger
// annotations live inside a struct body or inlined as a trailing comment, so
// the godoc above each declaration stays clean and human-facing.
package afterdecl

// snippet:struct

// Widget is a widget. This godoc stays clean — no swagger machinery here.
type Widget struct {
	// Widget is exposed to API consumers.
	//
	// swagger:model widgetModel
	// maxProperties: 5

	Name string `json:"name"`

	// Created is documented with a clean godoc; the format annotation is
	// inlined as a trailing comment.
	Created string `json:"created"` // swagger:strfmt date
}

// endsnippet:struct

// snippet:aliases

// Count is a plain count. Clean godoc above; annotation inlined below.
type Count int // swagger:model countType

// Stamp is a string alias. Clean godoc above; annotation inlined trailing.
type Stamp = string // swagger:model stampType

// endsnippet:aliases

// snippet:route

// ListWidgets is an ordinary Go handler with a clean godoc.
func ListWidgets() {
	// swagger:route GET /widgets widgets listWidgets
	//
	// Lists the widgets.
	//
	// Responses:
	//   200: widgetModel
}

// endsnippet:route
