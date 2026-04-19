// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package top_level_kinds declares top-level types of every Go kind
// (basic, array, slice, map, interface) annotated with swagger:model so
// that schemaBuilder.buildFromDecl walks each non-struct case branch.
package top_level_kinds

// MyInt is a top-level named basic integer type.
//
// swagger:model MyInt
type MyInt int

// MyArray is a top-level named array.
//
// swagger:model MyArray
type MyArray [10]string

// MySlice is a top-level named slice.
//
// swagger:model MySlice
type MySlice []string

// MyMap is a top-level named map.
//
// swagger:model MyMap
type MyMap map[string]int

// MyInterface is a top-level named interface.
//
// swagger:model MyInterface
type MyInterface interface {
	// Identify returns the name of this object.
	Identify() string
}

// IgnoredModel is annotated as a model but also carries swagger:ignore,
// so the sectionedParser flags it as ignored and buildFromDecl returns
// early via its `sp.ignored` branch.
//
// swagger:model IgnoredModel
// swagger:ignore
type IgnoredModel struct {
	Value int `json:"value"`
}
