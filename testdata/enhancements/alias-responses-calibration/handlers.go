// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package alias_responses_calibration

// AliasedTopHandler binds the top-level aliased response to an
// operation so the response shape is observable on a path.
//
// swagger:route GET /aliased-top calibration aliasedTopOp
//
// Responses:
//
//	200: aliasedTopResponse
func AliasedTopHandler() {}

// DirectHandler binds the control response to an operation.
//
// swagger:route POST /direct calibration directOp
//
// Responses:
//
//	200: directResponse
func DirectHandler() {}

// BodyAliasPlainHandler binds the unannotated body-alias response.
//
// swagger:route GET /body-alias-plain calibration bodyAliasPlainOp
//
// Responses:
//
//	200: bodyAliasPlainResponse
func BodyAliasPlainHandler() {}

// BodyAliasModeledHandler binds the annotated body-alias response.
//
// swagger:route GET /body-alias-modeled calibration bodyAliasModeledOp
//
// Responses:
//
//	200: bodyAliasModeledResponse
func BodyAliasModeledHandler() {}

// BodyAliasChainHandler binds the 2-link chain body-alias response.
//
// swagger:route GET /body-alias-chain calibration bodyAliasChainOp
//
// Responses:
//
//	200: bodyAliasChainResponse
func BodyAliasChainHandler() {}
