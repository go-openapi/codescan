// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Command genspec scans annotated Go source and writes the Swagger 2.0 specification it describes.
//
//	go install github.com/go-openapi/codescan/cmd/genspec@latest
//	genspec -workdir ./api ./...
//
// It is the ordinary, native way to run codescan: everything the library can be told is a flag, the
// document goes to standard output or to -output, and what the scan observed goes to standard error
// as colored diagnostics. See -h for the whole surface.
//
// # The family
//
// Three commands run the same scan, and which one to reach for is a question about the machine
// rather than about the specification:
//
//	genspec       here. Native, colored diagnostics, YAML, merges into an existing document,
//	              validates what it produced
//	genspec-wasi  the same scan with no dependency beyond the library, so it cross-compiles to
//	              WebAssembly and runs under a WASI runtime with no Go toolchain installed. It also
//	              speaks a machine-readable envelope (-format=json) carrying diagnostics and
//	              cross-references
//	genspec-tui   the same scan, live, with the source and the document side by side
//
// # Configuration file
//
// Anything that can be a flag can be preset in a .codescan.yaml, found by searching upwards from
// wherever the command is run. Keys are grouped into sections, and inside a section a key is the
// flag it sets, spelled exactly as on the command line:
//
//	scan:
//	  workdir: ./api
//	emit:
//	  scan-models: true
//	document:
//	  format: yaml
//
// Anything typed on the command line wins over the file, including a flag typed with the value it
// already had. -config names a particular file; -config off ignores whatever is lying around.
//
// One file serves the whole family: a section a command does not know is skipped rather than
// refused, so another command's settings may sit beside these. A key inside a section it does know
// must name one of its flags, which is what makes a typo an error rather than a setting that quietly
// never applied.
//
// # Relationship to go-swagger
//
// genspec does the same job as go-swagger's `swagger generate spec`, which drives the same library.
// It is released on its own, so fixes and enhancements reach it at codescan's pace rather than
// go-swagger's; go-swagger has a wider scope, and the dependencies that come with it.
//
// # Exit status
//
// A specification is written whenever one could be produced, so a non-zero status describes what was
// wrong with it rather than meaning nothing came out:
//
//	0  the scan produced a document, and nothing asked for more
//	1  the scan failed
//	2  the command line does not make sense
//	3  what was reported reached the severity -fail-on names
//	4  -validate found the document invalid
package main
