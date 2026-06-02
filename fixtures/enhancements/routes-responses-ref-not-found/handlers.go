// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_ref_not_found exercises the not-found branch
// of the legacy parseResponseLine fallback (lines 83-87). When an
// untagged response name is missing from BOTH the responses map AND
// the definitions map, the legacy parser silently emits a dangling
// `$ref: #/responses/<name>` that points at nothing.
//
// QUIRK: dangling refs are not flagged. Companion of
// routes-responses-definition-fallback which exercises the found
// branch. Together they witness both halves of the fallback logic.
package routes_responses_ref_not_found

// ListItems swagger:route GET /items items listItemsRefNotFound
//
// List items.
//
// Responses:
//
//	200: doesNotExistAnywhere
func ListItems() {}
