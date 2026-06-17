// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package mixed is a name-identity corpus member exercising BOTH collision
// kinds in one tree:
//
//   - EXPLICIT collision: x.Item and y.Item are each `swagger:model Item`.
//   - IMPLICIT collision: x.Record and y.Record are plain (unannotated) struct
//     types, discovered only because Root references them; they collide on the
//     auto-derived key "Record".
//
// Today all four collapse pairwise into "Item" and "Record". Target: four
// distinct definitions, with reduce treating explicit and implicit names
// identically (name-source-agnostic).
package mixed

import (
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-mixed/x"
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-mixed/y"
)

// swagger:model Root
type Root struct {
	EItem1 x.Item   `json:"eitem1"`
	EItem2 y.Item   `json:"eitem2"`
	IRec1  x.Record `json:"irec1"`
	IRec2  y.Record `json:"irec2"`
}
