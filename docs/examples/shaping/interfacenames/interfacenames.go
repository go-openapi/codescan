// SPDX-License-Identifier: Apache-2.0

// Package interfacenames is the test-backed witness for the "Interface-method
// property names" how-to. An interface method has no natural JSON
// serialization, so codescan invents a property name by running the
// swag/mangling transform on the Go method name (ID → id, CreatedAt →
// createdAt). It is scanned with SkipJSONifyInterfaceMethods off (the mangler
// runs) and on (the Go method name is emitted verbatim). A swagger:name
// override is taken verbatim either way.
package interfacenames

// snippet:account

// Account is a read model whose shape is described by interface methods.
//
// swagger:model Account
type Account interface {
	// ID is emitted as "id" by default; verbatim "ID" with the opt-out set.
	ID() string

	// CreatedAt is emitted as "createdAt" by default; verbatim "CreatedAt" with
	// the opt-out set.
	CreatedAt() string

	// swagger:name explicit_name
	//
	// A swagger:name override is taken verbatim either way — re-mangling would
	// camelCase it to "explicitName".
	OverriddenField() string
}

// endsnippet:account
