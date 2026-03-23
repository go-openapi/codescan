// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package duplicate_body_tag declares a swagger:route whose Responses
// block has `body:` appearing twice so parseTags trips its "valid tag
// but not in a valid position" error branch.
package duplicate_body_tag

// DoBad swagger:route GET /duplicate-body bad doDupBody
//
// Responses:
//
//	200: body:Foo body:Bar
func DoBad() {}
