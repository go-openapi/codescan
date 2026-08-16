// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2020

// MULTIPLE swagger:parameters structs in one type(...) block (go-swagger#2020):
// each is detected and bound to its operation by id.
type (
	// swagger:parameters alphaOp
	AlphaParams struct {
		// in: query
		Q string `json:"q"`
	}

	// swagger:parameters betaOp
	BetaParams struct {
		// in: query
		R string `json:"r"`
	}
)

// swagger:route GET /alpha alpha alphaOp
//
// Alpha.
//
// responses:
//
//	200: description: ok
func Alpha() {}

// swagger:route GET /beta beta betaOp
//
// Beta.
//
// responses:
//
//	200: description: ok
func Beta() {}
