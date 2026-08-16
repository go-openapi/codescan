// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package embedinheritance exercises the generalized "annotations on an
// embedded field apply to the members it promotes" rule (go-swagger#2701)
// beyond parameters: a `required:` annotation on an embed propagates to the
// promoted properties of a model schema, and of a response body schema
// (which is built through the same schema builder). Exportedness stays
// per-field: unexported promoted members never surface, at any depth.
package embedinheritance

// Base carries the fields promoted into its embedders.
type Base struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	hidden string // unexported → never rendered
}

// Widget embeds Base with a `required:` annotation on the embed, so the
// promoted id/name become required on Widget.
//
// swagger:model Widget
type Widget struct {
	// required: true
	Base

	// Extra is a normal field that is NOT required.
	Extra string `json:"extra"`
}

// envelope is the response body payload; it embeds Base as required too.
type envelope struct {
	// required: true
	Base
}

// swagger:response widgetResponse
type widgetResponse struct {
	// in: body
	Body envelope
}

// swagger:route GET /widget getWidget
//
// responses:
//   200: widgetResponse
func handler() {}

type headerSet struct {
	XRequestID string `json:"X-Request-Id"`
	XCount     int    `json:"X-Count"`
}

// swagger:response embeddedHeadersResponse
type embeddedHeadersResponse struct {
	// in: header
	headerSet
}

type bodyPayload struct {
	Message string `json:"message"`
}

// embeddedBodyResponse anonymously embeds a struct marked `in: body`. The
// embed IS the response body — the body $refs the embedded struct rather than
// promoting its fields as headers, exactly like a named `Body` field
// (go-swagger#1635, the responses counterpart of the in: embed inheritance).
//
// swagger:response embeddedBodyResponse
type embeddedBodyResponse struct {
	// in: body
	bodyPayload
}
