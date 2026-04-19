// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package unknown_annotation carries a deliberately bogus swagger
// annotation so the classifier returns an error from detectNodes.
package unknown_annotation

// Bogus uses an unknown swagger annotation.
//
// swagger:doesnotexist BogusTag
type Bogus struct {
	ID int64 `json:"id"`
}
