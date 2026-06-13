// SPDX-License-Identifier: Apache-2.0

// Package buildtags holds the annotated declarations used by the "Build tags"
// how-to. buildtags_test.go scans it without and with the "experimental" build
// tag and writes the before/after golden fragments.
package buildtags

// snippet:stable

// Stable is always scanned.
//
// swagger:model
type Stable struct {
	// Name is the feature name.
	Name string `json:"name"`
}

// endsnippet:stable
