// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package toplevelexample covers example: on top-level (non-struct) response
// body types — array and scalar — across placements (go-swagger#3013).
package toplevelexample

// ArrayResp is an array response with the example on its own paragraph.
//
// example: ["10.10.10.10","20.20.20.20"]
//
// swagger:response ArrayResp
type ArrayResp []string

// ScalarResp is a scalar (string) response with a trailing example line.
//
// swagger:response ScalarResp
// example: hello
type ScalarResp string

// swagger:route GET /array things getArray
//
// responses:
//   200: ArrayResp

// swagger:route GET /scalar things getScalar
//
// responses:
//   200: ScalarResp
func handlers2() {}
