// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug439

import "time"

// Unrelated is an unannotated type with a fancy field — the old engine scanned
// it too deeply and errored (#439). It must be left alone.
type Unrelated struct {
	When time.Time
	Ch   chan int
}

// BodyParamType is the actual body.
//
// swagger:model
type BodyParamType struct {
	A string `json:"a"`
	B int    `json:"b"`
}

// swagger:parameters myOperation
type paramType struct {
	// in: body
	Body BodyParamType
}

// swagger:route POST /op op myOperation
//
// Op.
//
// responses:
//
//	200: description: ok
func MyOperation() {}
