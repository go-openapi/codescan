// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package alias_parameters_calibration

// AliasedTopHandler binds the top-level aliased parameter set to an
// operation. Without a route, `swagger:parameters aliasedTop` would
// be registered but never merged into an operation, leaving `paths`
// empty in the captured spec and hiding the body/query semantics
// the workshop needs to judge.
//
// swagger:route GET /aliased-top calibration aliasedTop
//
// Responses:
//
//	200: description: ok
func AliasedTopHandler() {}

// DirectHandler binds the directly-declared parameter set (the
// control case) to its operation.
//
// swagger:route POST /direct calibration directParams
//
// Responses:
//
//	200: description: ok
func DirectHandler() {}
