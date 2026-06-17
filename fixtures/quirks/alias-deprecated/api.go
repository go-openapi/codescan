// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package aliasdeprecated is the F8 witness (doc-site-quirks.md): swagger:alias
// is deprecated. It once force-inlined a named primitive ({type:string})
// instead of the $ref a named type otherwise gets; that behaviour is removed.
// The annotation is now an empty sink that only emits a deprecation diagnostic;
// the type follows default handling.
package aliasdeprecated

// Email is a string refinement type carrying the deprecated swagger:alias
// annotation alongside swagger:model.
//
// swagger:alias
// swagger:model Email
type Email string

// Account references the deprecated-alias type.
//
// swagger:model Account
type Account struct {
	// Contact is an Email; it now resolves to a $ref, not an inlined string.
	Contact Email `json:"contact"`
}
