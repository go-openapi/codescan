// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2547

// Thing probes string example/default values written with surrounding quotes
// (go-swagger#2547, quirk F8). The quotes are delimiters and must be stripped:
// `example: ""` is the empty string, `example: "Foo"` is `Foo`.
//
// swagger:model
type Thing struct {
	// the label
	// example: ""
	Label string `json:"label"`

	// the title
	// example: "Foo"
	// default: "bar"
	Title string `json:"title"`
}
