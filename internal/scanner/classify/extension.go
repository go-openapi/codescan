// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package classify provides small classification predicates used by
// the scanner and by builders to decide whether a given name or
// comment line belongs to a particular Swagger-annotation family.
//
// The package lives beneath internal/scanner/ because classification
// is fundamentally a scanner concern: "does this string denote a
// swagger:xxx construct?" is the same kind of question the scanner
// asks when indexing packages. Builders that need the same predicate
// (vendor-extension key filtering, for instance) import from here
// rather than reaching back into internal/parsers/.
package classify

import "regexp"

// rxAllowedExtension matches a Swagger vendor-extension key:
// a leading `x-` or `X-` followed by at least one character.
var rxAllowedExtension = regexp.MustCompile(`^[Xx]-`)

// IsAllowedExtension reports whether key is a valid Swagger
// vendor-extension key ("x-..." / "X-...").
func IsAllowedExtension(key string) bool {
	return rxAllowedExtension.MatchString(key)
}
