// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package response_header_ref_leak pins Q2's fix on the response
// side: a response-header field whose Go type triggers the schema
// builder's makeRef path no longer corrupts `response.Schema` with
// a $ref. OAS v2 forbids $ref on response headers entirely
// (headers are SimpleSchema, no model references allowed); the
// pre-Q2 `responseTypable.SetRef` blindly wrote `response.Schema.Ref`
// regardless of the typable's `in`. Q2 changes SetRef to no-op
// under non-body mode and flags the attempt via a `refAttempted`
// state so the SimpleSchema exit validator's HasRef probe catches
// it and emits CodeUnsupportedInSimpleSchema.
//
// Two response shapes captured by the integration test golden:
//
//   - LeakResponse — header typed as a named struct, no strfmt.
//     Post-fix the response carries only the header entry (still
//     empty, since the named struct can't reduce to a SimpleSchema
//     primitive); no body Schema; diagnostic fires.
//
//   - LeakWithStrfmtResponse — same shape plus `swagger:strfmt
//     uuid` on the field. Post-fix the header is {string, uuid}
//     (the strfmt override fires after the exit-validator reset);
//     no body Schema; the diagnostic still surfaces for the
//     underlying ref attempt.
package response_header_ref_leak

// Tag is a named struct intended to be referenceable as a model
// from body schemas. Using it as the type of a header field is the
// author misuse this fixture captures.
//
// swagger:model
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// LeakResponse pins the post-fix shape: the header field is
// surfaced, the body schema is NOT corrupted, and the diagnostic
// signals the misuse.
//
// swagger:response leakResponse
type LeakResponse struct {
	// TagHeader is a header field typed as a named struct. Pre-Q2
	// this leaked `$ref: "#/definitions/Tag"` onto resp.Schema and
	// left the header empty. Post-fix resp.Schema stays nil; the
	// header surfaces empty (no Type — the named struct can't
	// reduce to a SimpleSchema primitive); diagnostic fires.
	//
	// in: header
	TagHeader Tag `json:"X-Tag"`
}

// LeakWithStrfmtResponse pins the post-fix shape under the strfmt
// override: the header gets {string, uuid}, the body schema stays
// nil, and the diagnostic still fires for the underlying ref
// attempt.
//
// swagger:response leakWithStrfmtResponse
type LeakWithStrfmtResponse struct {
	// TagID is a header field with strfmt override on a named-struct
	// type. The strfmt override runs after the exit-validator's
	// reset, so the header surfaces as {string, uuid}. No body-schema
	// leak.
	//
	// in: header
	// swagger:strfmt uuid
	TagID Tag `json:"X-Tag-ID"`
}
