// SPDX-License-Identifier: Apache-2.0

// Package overrides is the test-backed witness for the "Overriding titles &
// descriptions" how-to. It shows swagger:title / swagger:description replacing
// the godoc-derived text so the Go-facing comment can diverge from the
// API-facing spec, across models, fields, $ref'd fields, and responses.
package overrides

// snippet:model

// Widget is the Go-facing widget doc, written for Go readers.
//
// It explains internal Go usage that should not leak into the API spec.
//
// swagger:model
// swagger:title A Public Widget
// swagger:description A widget exposed via the public API.
type Widget struct {
	// ID explains the Go field for Go readers.
	//
	// swagger:description The unique widget identifier.
	ID string `json:"id"`

	// Label is the Go-facing field doc. Fields carry no title by default;
	// the override is the only way a property gets one.
	//
	// swagger:title Display Label
	// swagger:description Human-readable label shown to API consumers.
	Label string `json:"label"`

	// Plain keeps its godoc description because it carries no override.
	Plain string `json:"plain"`

	// Capacity combines a description override with an inline validation
	// keyword on the same field: the override applies AND maximum is kept,
	// because the override annotations dispatch through the schema family.
	//
	// swagger:description The maximum capacity, in liters.
	// maximum: 1000
	Capacity int64 `json:"capacity"`

	// Suppressed has a godoc that a bare swagger:description suppresses: the
	// empty value is applied (description omitted) and scan.empty-override is
	// raised, in case the bare marker was left behind by mistake.
	//
	// swagger:description
	Suppressed string `json:"suppressed"`

	// Notes carries a multi-line description override: the lines following the
	// annotation fold into the description until the blank line, joined with
	// newlines.
	//
	// swagger:description Free-form notes about the widget.
	// They may span several lines, all folded into one description.
	//
	// The blank line above terminates the override body; this paragraph is
	// ordinary godoc and is discarded (the override won).
	Notes string `json:"notes"`

	// Gadget is a $ref field carrying title + description overrides. They are
	// symmetric $ref siblings: kept under EmitRefSiblings, dropped to a bare
	// $ref under the default flags — the same rule a prose description follows.
	//
	// swagger:title Gadget Ref
	// swagger:description The attached gadget, described for API consumers.
	Gadget Gadget `json:"gadget"`
}

// Gadget is a plain referenced model.
//
// swagger:model
type Gadget struct {
	Serial string `json:"serial"`
}

// endsnippet:model

// snippet:response

// ErrorResponse is the Go-facing response doc, written for Go readers — it
// should not leak into the API spec.
//
// swagger:response errorResponse
// swagger:description The error payload returned to API consumers.
// swagger:title Ignored — responses have no title
type ErrorResponse struct {
	// XErrorCode is the Go-facing header doc.
	//
	// swagger:description The machine-readable error code.
	XErrorCode string `json:"X-Error-Code"` //nolint:tagliatelle // canonical HTTP header name

	// ErrorBody carries the structured error.
	//
	// in: body
	Body ErrorBody `json:"body"`
}

// ErrorBody is the error payload returned in the response body.
//
// swagger:model
type ErrorBody struct {
	Message string `json:"message"`
}

// endsnippet:response
