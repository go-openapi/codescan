// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package godoclinkswalk

// CommonTrace registers a shared header parameter at the spec top level, so its prose lives under
// #/parameters rather than inside an operation.
//
// swagger:parameters *
type CommonTrace struct {
	// TraceID correlates a request touching a [Widget].
	//
	// in: header
	TraceID string `json:"X-Trace-ID"`
}

// WidgetPathParams are inlined into the /widgets path item, so their prose sits beside the
// operations rather than inside one.
//
// swagger:parameters /widgets
type WidgetPathParams struct {
	// Tenant scopes the [Widget] listing.
	//
	// in: header
	Tenant string `json:"X-Tenant"`
}

// WidgetError is force-registered at #/responses, so its prose lives in the spec-level response
// namespace.
//
// swagger:response *
type WidgetError struct {
	// in: body
	Body struct {
		// Message explains why the [Widget] could not be served.
		Message string `json:"message"`
	} `json:"body"`
}

// WidgetOK returns a [Widget], and carries a header whose prose has to be walked too.
//
// swagger:response widgetOK
type WidgetOK struct {
	// ETag versions the returned [Widget].
	ETag string `json:"ETag"`

	// in: body
	Body Widget `json:"body"`
}

// ListWidgets lists widgets.
//
// swagger:route GET /widgets widgets listWidgets
//
// Lists every [Widget] on offer.
//
// The listing names each [Gadget] fitted to it.
//
// Parameters:
//   + name: since
//     in: query
//     description: only list a [Widget] created after this time
//     type: string
//
// Responses:
//
//	200: widgetOK
//	default: WidgetError
func ListWidgets() {}
