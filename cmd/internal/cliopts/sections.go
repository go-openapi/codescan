// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliopts

import "github.com/go-openapi/codescan/cmd/internal/cliconf"

// The sections a configuration file addresses these flags in.
//
// They are the four questions a scan answers, in the order it answers them: which code, built how,
// read how, rendered how. The same grouping the command help and the READMEs use - a reader who has
// learnt one has learnt the others.
const (
	// sectionScan is which code is looked at.
	sectionScan = "scan"
	// sectionGo is what it is built as: the go environment that decides what compiles.
	sectionGo = "go"
	// sectionLoad is how the packages are read.
	sectionLoad = "load"
	// sectionEmit is what the specification ends up saying.
	sectionEmit = "emit"
)

// ConfigSchema reports the section each of these flags is addressed in.
//
// A command merges this with its own flags' sections to get the schema a configuration file is
// checked against. Built from the tables rather than written out again, so a flag cannot arrive with
// no way to configure it.
func ConfigSchema() cliconf.Schema {
	schema := make(cliconf.Schema, len(boolOptions)+len(stringOptions)+len(floatOptions)+len(listOptions)+1)

	for _, opt := range boolOptions {
		schema[opt.name] = opt.section
	}
	for _, opt := range stringOptions {
		schema[opt.name] = opt.section
	}
	for _, opt := range floatOptions {
		schema[opt.name] = opt.section
	}
	for _, opt := range listOptions {
		schema[opt.name] = opt.section
	}
	schema[loaderFlag] = sectionLoad

	return schema
}
