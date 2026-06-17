// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package singlelinedesc exercises the SingleLineCommentAsDescription
// option: a one-line doc comment resolves to the model description instead
// of its title, while a multi-line comment keeps the title/description
// split.
package singlelinedesc

// Widget is a one-line model comment ending in a period.
//
// swagger:model
type Widget struct {
	Name string `json:"name"`
}

// Gadget is the title line.
//
// And this is the multi-line description body.
//
// swagger:model
type Gadget struct {
	Name string `json:"name"`
}
