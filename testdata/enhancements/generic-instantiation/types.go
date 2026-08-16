// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package generic_instantiation exercises the generic-type
// instantiation short-circuit inside [Builder.buildNamedType]:
// when a named type has TypeArgs (i.e. is an instantiation of a
// generic declaration like `GenericSlice[int]`), we unwrap to its
// parameterised Underlying instead of trying to $ref to it.
//
// Without the short-circuit, an instantiated field type would route
// through resolveRefOr → makeRef → $ref to the generic declaration,
// which itself emits as an empty schema (its type parameters are
// filtered as UnsupportedBuiltinType). With the short-circuit, the
// field correctly reflects the substituted underlying.
package generic_instantiation

// GenericSlice is the generic declaration. Its own emitted schema
// is essentially empty because the type parameter `T` is filtered
// out by UnsupportedBuiltinType.
type GenericSlice[T any] []T

// GenericMap is a two-parameter generic declaration.
type GenericMap[K comparable, V any] map[K]V

// Container holds fields whose types are generic INSTANTIATIONS.
// These should emit with the substituted underlying shape, not as
// $ref to the (empty) generic declaration.
//
// swagger:model generic_container
type Container struct {
	// Items is an instantiation of GenericSlice with T=int.
	// Expected schema: {array, items:{integer, int64}}.
	Items GenericSlice[int] `json:"items"`

	// Names is an instantiation of GenericSlice with T=string.
	// Expected schema: {array, items:{string}}.
	Names GenericSlice[string] `json:"names"`

	// Counts is an instantiation of GenericMap with K=string, V=int.
	// Expected schema: {object, additionalProperties:{integer, int64}}.
	Counts GenericMap[string, int] `json:"counts"`
}
