// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package cliconf supplements a command line with a configuration file.
//
// It owns the contract - where the file is, what may be in it, and how it loses to anything the
// caller typed - and nothing else. The values themselves arrive as a plain map, so a command is free
// to read them however it likes: through koanf, which brings environment variables and further
// formats with it, or through [Parse], which needs nothing this repository does not already have.
// That seam is why this package can be shared with a command that must cross-compile to
// WebAssembly with no dependency beyond the library.
//
// # What a file looks like
//
// Keys are grouped into sections, and within a section a key is the flag it sets, spelled exactly as
// on the command line:
//
//	scan:
//	  workdir: ./api
//	  exclude-tags: [internal, debug]
//	emit:
//	  scan-models: true
//	  name-from-tags: [form, json]
//
// Sections are how one file serves several commands: a section this command does not know is
// skipped rather than refused, and reported through [Result] so it can be mentioned rather than
// silently dropped. A key inside a section it does know must name one of its flags - that is what
// makes a typo an error rather than a setting that quietly never applied.
//
// # Precedence
//
// A flag the caller typed always wins, whatever the file says. That is decided by asking the flag
// set which flags were actually seen, so it holds for a flag set to its own default value too:
// -scan-models=false means false, even where the file says true.
//
// Everything else lands through [flag.FlagSet.Set], the same path the command line takes, so a value
// is parsed and validated exactly once and a file cannot express something an argument could not.
package cliconf
