// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package interface_no_mangle exercises Options.SkipJSONifyInterfaceMethods —
// the opt-out of the interface-method auto-jsonify mangler.
//
// An interface method has no natural JSON serialization, so by default codescan
// runs the swag/mangling ToJSONName transform on the Go method name to invent a
// property name (ID -> id, CreatedAt -> createdAt). With the opt-out set the Go
// method name is emitted verbatim instead. A swagger:name override is always
// taken verbatim, independent of the flag.
//
// The default-path methods deliberately mangle to a different spelling than
// their Go name so the on/off difference is observable; the override method
// pins the contract that an explicit name always wins, regardless of the flag.
//
// See internal/builders/schema/README.md §interface-naming.
package interface_no_mangle

// Account is a no-mangle witness interface.
//
// swagger:model Account
type Account interface {
	// Default path, no override. The mangler produces "id"; with the opt-out
	// the verbatim "ID" is emitted.
	ID() string

	// Default path, no override. The mangler produces "createdAt"; with the
	// opt-out the verbatim "CreatedAt" is emitted.
	CreatedAt() string

	// swagger:name explicit_name
	//
	// A swagger:name override is taken verbatim whether or not the mangler is
	// skipped — re-mangling would camelCase this to "explicitName".
	OverriddenField() string
}
