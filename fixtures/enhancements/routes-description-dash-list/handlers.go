// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_description_dash_list witnesses the post-M6.5-C
// behaviour for route description lines that begin with `-` (a
// markdown list item). Before the parsePathAnnotation /
// trimCommentPrefix cleanup, routes' description handler would
// silently strip the leading `-` from every prose line. Now grammar's
// lexer's `//` strip path (trimContentPrefix) runs over each
// synthetic per-line comment, and only ` \t*/|` are stripped — `-`
// survives, matching how every other annotation handles prose.
package routes_description_dash_list

/* CreateThing swagger:route POST /things things createThing

Create a thing.

This endpoint:
- accepts a thing payload
- returns 201 on success
- returns a problem document on failure

Responses:
  200: description: OK
*/
func CreateThing() {}
