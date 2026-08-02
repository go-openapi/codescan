// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package route_name_shapes witnesses the name shapes a path annotation accepts,
// and what happens to one it cannot parse.
//
// A tag and an operationId of a SINGLE character used to void the entire
// annotation. The failure was not local to the short name: the tags group is
// optional, so the parse fell back to matching with no tags at all, leaving the
// operationId pattern to swallow `e listOne` — which its alphabet has no space
// for. The line then matched nothing, and a swagger:route matching nothing is
// not a malformed route, it is not a route, so no diagnostic was possible and
// the path simply never appeared.
package route_name_shapes

// HandlerShortTag has a one-character tag.
//
// swagger:route GET /short-tag e listOne
//
// Responses:
//
//	200: emptyResp
func HandlerShortTag() {}

// HandlerShortID has a one-character operationId.
//
// swagger:route GET /short-id shapes l
//
// Responses:
//
//	200: emptyResp
func HandlerShortID() {}

// HandlerShortBoth has both, which is where the fallback used to bite hardest.
//
// swagger:route GET /short-both e l
//
// Responses:
//
//	200: emptyResp
func HandlerShortBoth() {}

// HandlerShortIDNoTags has a one-character operationId and no tags at all.
//
// swagger:route GET /short-id-no-tags q
//
// Responses:
//
//	200: emptyResp
func HandlerShortIDNoTags() {}

// HandlerShortAmongTags carries a one-character tag beside a longer one.
//
// swagger:route GET /short-among a shapes listAmong
//
// Responses:
//
//	200: emptyResp
func HandlerShortAmongTags() {}

// EmptyResp is the shared response body.
//
// swagger:response emptyResp
type EmptyResp struct {
	// in: body
	Body string `json:"body"`
}
