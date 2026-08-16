// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package response_edges exercises edge cases in the response builder:
// embedded response structs, time.Time header fields, strfmt-tagged named
// header fields, and top-level non-struct response payloads.
package response_edges

import "time"

// RequestID is a named string with a swagger:strfmt tag used as a header
// field so that buildNamedField takes its strfmt branch.
//
// swagger:strfmt uuid
type RequestID string

// CommonHeaders is embedded into the full response so that
// responseBuilder.buildFromStruct walks the embedded-field branch and
// recursively invokes buildFromType on an embedded named type.
type CommonHeaders struct {
	// The request trace identifier.
	//
	// in: header
	TraceID RequestID `json:"X-Trace-ID"`
}

// Body is the canonical body for the full response.
//
// swagger:model Body
type Body struct {
	// required: true
	ID int64 `json:"id"`

	Payload string `json:"payload"`
}

// FullResponse carries headers (embedded + inline) and a body field so
// that buildFromStruct, processResponseField and buildNamedField are all
// exercised in a single scan.
//
// swagger:response fullResponse
type FullResponse struct {
	CommonHeaders

	// The request rate-limit window.
	//
	// in: header
	RateLimit int `json:"X-Rate-Limit"`

	// The server-side timestamp for this response.
	//
	// in: header
	Timestamp time.Time `json:"X-Timestamp"`

	// in: body
	Body Body `json:"body"`
}

// IDs is a named slice so that responseBuilder.buildNamedType walks its
// non-struct default branch.
//
// swagger:response idsResponse
type IDs []int64
