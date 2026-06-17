// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package gofmtmeta is the F7 quirk witness: a swagger:meta block in its
// gofmt-CANONICAL shape. A user writes a column-0 YAML key with indented
// children; gofmt then inserts the blank "//" line below (its doc-comment
// code-block rule), producing exactly the form below. The meta YAML parser
// once rejected it with "found character that cannot start any token" because
// the tab indentation after the blank line reached the YAML parser; it now
// dedents off the first non-blank line and parses the block identically to
// the uniformly-indented form.
//
// SecurityDefinitions:
//
//	oauth2:
//		type: oauth2
//		in: header
//		flow: application
//
// swagger:meta
package gofmtmeta
