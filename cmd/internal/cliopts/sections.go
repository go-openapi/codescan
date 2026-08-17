// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliopts

import (
	"maps"

	"github.com/go-openapi/codescan/cmd/internal/cliconf"
)

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
// notConfigurable are the shared options a configuration file may not address, with the reason.
//
// A configuration file is found by searching upwards, so running a command inside a repository reads
// THAT repository's file. Most of what a file sets shapes the document, which is what one is for.
// An option that decides a PATH is different: it would let the tree being scanned choose where the
// command reads or writes, and the tree is somebody else's.
//
// Command line only, so the answer is always the one the person running the command typed.
var notConfigurable = map[string]string{ //nolint:gochecknoglobals // the table, read once at startup
	"workdir": "it decides where the scan runs, so a discovered file could point the scan at a tree " +
		"nobody asked about",
}

// NotConfigurable reports the shared options a file may not address, so a command can excuse them in
// the guard that checks every flag is either addressable or deliberately not.
func NotConfigurable() map[string]string { return maps.Clone(notConfigurable) }

// ConfigSchema is where each shared option is addressed in a configuration file.
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

	for name := range notConfigurable {
		delete(schema, name)
	}

	return schema
}
