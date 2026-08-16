// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2942 reproduces go-swagger issue #2942 ("how can i generate spec
// with a simple string response"). Two paths to a primitive response body:
//   - a swagger:route response line using the unambiguous `body:` tag
//     (`200: body:string`); and
//   - a swagger:response whose Body field is a primitive (`Body string`).
//
// The bare/untagged form (`200: string`) is intentionally NOT supported — an
// untagged token is a response/model NAME, and reading it as a type would make
// the syntax ambiguous; authors are pointed at the `body:` form by diagnostic.
package bug2942

// swagger:route GET /version getVersion
//
// responses:
//   200: body:string

// versionResponse wraps a primitive string body.
//
// swagger:response versionResponse
type versionResponse struct {
	// in: body
	Body string
}

// swagger:route GET /version2 getVersion2
//
// responses:
//
//	200: versionResponse
func h() {}
