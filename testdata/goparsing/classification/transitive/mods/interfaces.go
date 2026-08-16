// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mods

// ExtraInfo is an interface for things that have extra info
// swagger:model extra
type ExtraInfo interface {
	// swagger:name extraInfo
	ExtraInfo() string
}

// EmbeddedColor is a color
//
// swagger:model color
type EmbeddedColor interface {
	// swagger:name colorName
	ColorName() string
}
