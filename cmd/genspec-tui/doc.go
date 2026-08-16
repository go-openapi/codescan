// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Command genspec-tui is an interactive terminal front-end for the codescan Swagger-spec generator.
//
// It provides a source-tree browser (left), the generated spec (right, JSON or YAML), and diagnostics (bottom).
//
// It regenerates the whole-scope spec on any file change.
//
// The scan is configured from two places.
//   - Boolean knobs are toggled live in the options overlay (o)
//   - build tags, package and tag filters, naming - are command-line flags (a checkbox list cannot express them)
//
// The knobs settings overlay re-runs the scan on close.
//
// Defaults at startup may be consumed from a configuration file or environment variables.
package main
