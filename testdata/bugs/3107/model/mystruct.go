// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package model reproduces go-swagger issue #3107 ("No struct definition
// in swagger generate"): under `generate spec -m`, MyStruct was emitted as
// an empty definition carrying only `x-go-package`, with its fields
// dropped. The model lives in its own package, referenced cross-package by
// the api package's response body, so the fix must resolve the type
// through the import graph rather than only from directly-scanned files.
package model

// swagger:model MyStruct
type MyStruct struct {
	Field1 string `json:"field1" binding:"required"`
	Field2 string `json:"field2" binding:"required"`
}
