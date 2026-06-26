// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package inner_markdown exercises the swagger:description | literal block
// scalar (reframes go-swagger#3211): an author opts a description into verbatim
// markdown by ending the annotation line with the YAML literal-block marker `|`.
// The body below is captured exactly — blank lines, indentation, and table
// pipes preserved — until the next swagger annotation or end of comment.
package inner_markdown

// Widget is a thing.
//
// swagger:description |
// Widgets support **markdown** in their description:
//
// | name | purpose |
// |------|---------|
// | foo  | bars    |
//
// - point one
// - point two
//
// swagger:model Widget
type Widget struct {
	// Name of the widget.
	//
	// swagger:description |
	// The name must be:
	//
	//   1. unique
	//   2. lowercase
	Name string `json:"name"`
}
