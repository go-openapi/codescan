// SPDX-License-Identifier: Apache-2.0

//go:build experimental

package buildtags

// snippet:experimental

// Experimental is only scanned when the "experimental" build tag is set.
//
// swagger:model
type Experimental struct {
	// Beta flags a beta-only feature.
	Beta bool `json:"beta"`
}

// endsnippet:experimental
