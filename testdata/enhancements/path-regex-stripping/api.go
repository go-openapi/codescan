// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package pathregex exercises inline-regex path-parameter stripping across
// both the swagger:route and swagger:operation builders. OpenAPI 2.0 path
// templating follows RFC 6570 URI Template Level-1 expansion only, so any
// `{name:regex}` constraint is stripped to the bare `{name}` form (with a
// warning) rather than dropping the route silently.
package pathregex

// swagger:route GET /a/{x:[0-9]+}/b/{y:[a-z-]+} things multiParam
//
// responses:
//
//	200: description: ok
func multiParam() {}

// PlainRoute is the control: a bare RFC 6570 Level-1 template, no regex, so
// nothing is stripped and no warning is raised.
//
// swagger:route GET /plain/{id} things plainRoute
//
// responses:
//
//	200: description: ok
func plainRoute() {}

// getCode reaches the operation builder with a nested-brace quantifier.
//
// swagger:operation GET /codes/{code:[0-9]{2,4}} things getCode
//
// ---
// responses:
//   "200":
//     description: ok
func getCode() {}
