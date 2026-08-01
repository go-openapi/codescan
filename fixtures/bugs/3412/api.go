// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug3412 reproduces go-swagger issue #3412 ("`swagger:enum` seems to
// ignore negative integer constants"): Go never scans a negative numeric
// literal — `-1` is a unary minus applied to `1` — so a signed const reached
// the enum collector as an *ast.UnaryExpr and was silently dropped, leaving
// [0] where [-1, 0, 1] was declared.
package bug3412

// PanDirection is the direction of a pan.
//
// swagger:enum PanDirection
type PanDirection int8

const (
	// PanLeft pans to the left.
	PanLeft PanDirection = -1

	// NoPan does not pan.
	NoPan PanDirection = 0

	// PanRight pans to the right.
	PanRight PanDirection = +1
)

// ControlParams are PTZ control parameters.
//
// swagger:model ControlParams
type ControlParams struct {
	// specifies the direction of the pan.
	Pan PanDirection `json:"pan,omitempty"`
}
