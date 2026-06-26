// SPDX-License-Identifier: Apache-2.0

// Package markdowndesc is the test-backed witness for the "Markdown
// descriptions" how-to. It contrasts an ordinary description annotation (folded
// with the Option B rule, which stops at the first blank line) with the
// swagger:description | literal block-scalar marker, which captures the body
// verbatim — blank lines, indentation and table pipes preserved — until the
// next annotation or end of comment. Reframes go-swagger#3211.
package markdowndesc

// snippet:plain

// Plain uses an ordinary description annotation.
//
// swagger:description
// Option B folds prose up to the first blank line, so only this sentence
// survives — the markdown table below never reaches the spec.
//
// | name | purpose |
// |------|---------|
// | foo  | bars    |
//
// swagger:model Plain
type Plain struct {
	Name string `json:"name"`
}

// endsnippet:plain

// snippet:markdown

// Markdown opts into a verbatim body with the literal block marker.
//
// swagger:description |
// The body is captured **verbatim** — pipes, blank lines and all:
//
// | name | purpose |
// |------|---------|
// | foo  | bars    |
//
// - point one
// - point two
//
// swagger:model Markdown
type Markdown struct {
	// Name of the widget.
	//
	// swagger:description |
	// The name must be:
	//
	//   1. unique
	//   2. lowercase
	Name string `json:"name"`
}

// endsnippet:markdown
