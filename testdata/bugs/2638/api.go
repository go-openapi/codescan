// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2638 reproduces go-swagger issue #2638 ("Improper handling of
// multiple variables on one line"): a struct field group declaring several
// names on one line (`R, G, B, A uint8`) must emit one property per name,
// instead of a single mislabeled property.
package bug2638

// RGBA mirrors the upstream color.RGBA shape from the issue: four names share
// one field group.
//
// swagger:model RGBA
type RGBA struct {
	R, G, B, A uint8
}

// Mixed exercises a multi-name group alongside single-name fields, and a
// multi-name group that carries json options: the `,omitempty` applies to
// every member while the rename is dropped (a single rename can't name N
// members).
//
// swagger:model Mixed
type Mixed struct {
	X, Y, Z float64 `json:"renamed,omitempty"`
	Label   string
}
