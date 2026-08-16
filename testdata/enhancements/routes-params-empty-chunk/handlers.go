// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_empty_chunk exercises a bare `+` chunk with
// no following key:value lines. Legacy SetOpParams.Parse appends an
// empty parameter object.
package routes_params_empty_chunk

// EmptyChunk swagger:route GET /empty items emptyChunkOp
//
// Endpoint with an empty parameter chunk.
//
// Parameters:
//   +
//
// Responses:
//
//	200: description: OK
func EmptyChunk() {}
