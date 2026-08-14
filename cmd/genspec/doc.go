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
