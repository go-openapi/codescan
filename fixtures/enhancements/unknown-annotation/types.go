// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package unknown_annotation carries a deliberately bogus swagger annotation.
//
// The classifier reads it, cannot place it, and skips it with a warning. It used
// to abort the whole scan instead, so one mistyped keyword in one comment was
// enough to produce nothing at all from an entire package graph.
package unknown_annotation

// Bogus uses an unknown swagger annotation, and is emitted regardless: the
// comment carrying it is treated as prose.
//
// swagger:doesnotexist BogusTag
//
// swagger:model Bogus
type Bogus struct {
	ID int64 `json:"id"`
}
