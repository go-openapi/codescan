// SPDX-License-Identifier: Apache-2.0

// Package singleline is the witness for the "Single-line comments as
// descriptions" how-to. singleline_test.go scans it twice — with
// SingleLineCommentAsDescription off (default) and on — and writes the two
// goldens the guide compares, so the documentation tracks the scanner's real
// output.
package singleline

// snippet:model

// Gadget is a small device.
//
// swagger:model
type Gadget struct {
	Name string `json:"name"`
}

// endsnippet:model

// snippet:route

// swagger:route GET /gadgets gadgets listGadgets
//
// Lists every gadget in the catalog.
//
// responses:
//
//	200: gadgetsResponse

// endsnippet:route

// GadgetsResponse is the list payload.
//
// swagger:response gadgetsResponse
type GadgetsResponse struct {
	// in:body
	Body []Gadget
}
