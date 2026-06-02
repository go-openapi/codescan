// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package response_file_types pins Q3's response-side fix: the
// `swagger:file` annotation on a `swagger:response` field is gated
// on `in: body`. Pre-Q3 the file branch fired unconditionally and
// rewrote `resp.Schema` to `{file, ""}` even on header-positioned
// fields — silently corrupting the body schema from a misplaced
// annotation. Per OAS v2 the allowed header types are
// {string, number, integer, boolean, array}; `file` is forbidden
// on headers.
//
// Three response shapes pinned by the golden:
//
//   - FileBodyResponse — legitimate file body: `swagger:file` +
//     `in: body`. resp.Schema is {file, ""}; no headers.
//   - FileOnHeaderResponse — misuse: `swagger:file` + `in: header`
//     (or implicit default). Diagnostic emitted; the file branch
//     is skipped; the field falls through to the normal header
//     build and surfaces as a regular header.
//   - FileOnImplicitDefaultResponse — same misuse without the
//     `in:` line at all (Q1's implicit-header default applies).
//     Diagnostic still fires; field becomes a header.
package response_file_types

// FileBodyResponse is the legitimate case: the response IS a file.
// `swagger:file` on the Body field with `in: body` rewrites
// resp.Schema to {file, ""} and skips the field build.
//
// swagger:response fileBodyResponse
type FileBodyResponse struct {
	// File marks the response body as a file payload. Field shape
	// mirrors the canonical v1 file response: a `[]byte` field with
	// `in: body` + `swagger:file`.
	//
	// in: body
	// swagger:file
	File []byte
}

// FileOnHeaderResponse exercises the Q3 misuse: `swagger:file` on
// a header-positioned field. Post-fix the diagnostic fires and
// the field falls through to the normal build, surfacing as a
// regular header (typed as string here).
//
// swagger:response fileOnHeaderResponse
type FileOnHeaderResponse struct {
	// Misplaced has both `in: header` and `swagger:file`. The
	// scanner diagnoses and treats it as a normal header.
	//
	// in: header
	// swagger:file
	Misplaced string `json:"X-Misplaced"`
}

// FileOnImplicitDefaultResponse — `swagger:file` on a field with
// no `in:` line. After Q1 the implicit default is header, so this
// is also a misuse. Same diagnostic fires; field becomes a header.
//
// swagger:response fileOnImplicitDefaultResponse
type FileOnImplicitDefaultResponse struct {
	// Implicit has only `swagger:file` — no `in:` line at all.
	// Q1's implicit-header default applies; Q3's gate diagnoses.
	//
	// swagger:file
	Implicit string `json:"X-Implicit"`
}
