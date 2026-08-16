// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bundle names the dependencies the published export-data bundle covers.
//
// Nothing here is called. The imports exist so `go list` can resolve them, and so the set is a
// reviewable list in one place rather than an argument someone remembers to pass.
//
// What belongs here: the libraries an annotated API is likely to mention in a type the scanner has
// to render — a format, a stream, a spec type. What does not: anything a scan never sees.
package bundle

import (
	_ "github.com/go-openapi/runtime" // blank to trigger go list
	_ "github.com/go-openapi/spec"    // blank to trigger go list
	_ "github.com/go-openapi/strfmt"  // blank to trigger go list
	_ "github.com/go-openapi/swag"    // blank to trigger go list
)
