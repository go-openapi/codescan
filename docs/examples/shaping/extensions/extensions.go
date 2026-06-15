// SPDX-License-Identifier: Apache-2.0

// Package extensions holds the annotated declarations used by the
// "Vendor extensions" how-to. extensions_test.go scans it with SkipExtensions
// off and on and writes the before/after golden fragments.
package extensions

// snippet:model

// Widget is a small model.
//
// codescan records each field's Go origin as vendor extensions unless
// SkipExtensions is set.
//
// swagger:model
type Widget struct {
	// Label is the display label.
	Label string `json:"label"`

	// Size is the widget size in pixels.
	Size int32 `json:"size"`
}

// endsnippet:model

// snippet:paramext

// ListWidgetsParams decorates a query parameter with an author-supplied vendor
// extension through an Extensions: block — useful for tools (e.g. Dredd) that
// read x-example. A bare `x-example:` line would be swallowed as the
// description, so the Extensions: block is the supported form.
//
// swagger:parameters listWidgets
type ListWidgetsParams struct {
	// Page is the page number.
	//
	// in: query
	//
	// Extensions:
	//   x-example: 2
	Page int32 `json:"page"`
}

// WidgetList responds with a header that also carries a vendor extension —
// parameters and response headers both honour Extensions:.
//
// swagger:response widgetList
type WidgetList struct {
	// X-Rate-Limit is the per-window request budget.
	//
	// Extensions:
	//   x-units: requests-per-minute
	XRateLimit int32 `json:"X-Rate-Limit"` //nolint:tagliatelle // canonical HTTP header name
}

// swagger:route GET /widgets widgets listWidgets
//
// responses:
//
//	200: widgetList

// endsnippet:paramext
