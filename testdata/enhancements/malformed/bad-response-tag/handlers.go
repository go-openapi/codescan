// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bad_response_tag declares a swagger:route whose Responses
// block carries an unrecognised tag prefix, tripping parseTags's
// "invalid tag" branch.
package bad_response_tag

// DoBad swagger:route GET /bad-response bad doBadResp
//
// Responses:
//
//	200: notAValidTag: value
func DoBad() {}
