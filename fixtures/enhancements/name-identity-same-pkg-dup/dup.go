// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package samepkgdup is a name-identity corpus member for D-4: two DIFFERENT
// types in the SAME package both claim `swagger:model Dup`. This is a genuine
// user error (one package cannot own a name twice). Today they collide and
// merge. Target: keep the first, fall back to the goName for the duplicate
// (-> "Second") and emit a diagnostic.
package samepkgdup

// swagger:model Dup
type First struct {
	A string `json:"a"`
}

// swagger:model Dup
type Second struct {
	B int64 `json:"b"`
}

// Root references both so they are both discovered.
//
// swagger:model Root
type Root struct {
	F First  `json:"f"`
	S Second `json:"s"`
}
