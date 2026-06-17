// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package keywordxpkg witnesses cross-package leaf resolution for the
// type-name keyword sites (swagger:additionalProperties,
// swagger:patternProperties, swagger:type): a bare type name that lives in
// ANOTHER package should resolve by leaf against the discovered model set —
// uniquely (promote to its definition) or, when several packages share the
// leaf, with an ambiguity diagnostic + drop.
package keywordxpkg

import (
	"github.com/go-openapi/codescan/fixtures/enhancements/keyword-xpkg-leaf/a"
	"github.com/go-openapi/codescan/fixtures/enhancements/keyword-xpkg-leaf/b"
	"github.com/go-openapi/codescan/fixtures/enhancements/keyword-xpkg-leaf/dup1"
	"github.com/go-openapi/codescan/fixtures/enhancements/keyword-xpkg-leaf/dup2"
)

// keep the cross-package imports referenced so the packages load.
var (
	_ = a.Widget{}
	_ = b.Gadget{}
	_ = dup1.Thing{}
	_ = dup2.Thing{}
)

// Bag carries additionalProperties referencing a cross-package model by leaf
// (a.Widget). Unique leaf -> resolves to a $ref.
//
// swagger:model Bag
// swagger:additionalProperties Widget
type Bag struct {
	ID string `json:"id"`
}

// Catalog carries patternProperties referencing a cross-package model by leaf
// (a.Widget). Unique leaf -> resolves to a $ref.
//
// swagger:model Catalog
// swagger:patternProperties "^item-": Widget
type Catalog struct {
	ID string `json:"id"`
}

// AmbiguousBag references a leaf (Thing) declared as a model in TWO packages
// (dup1, dup2). Ambiguous -> diagnostic + the marker is dropped.
//
// swagger:model AmbiguousBag
// swagger:additionalProperties Thing
type AmbiguousBag struct {
	ID string `json:"id"`
}

// Holder overrides a field type with swagger:type referencing a cross-package
// model by leaf (b.Gadget). swagger:type inlines, so the field becomes the
// Gadget structure in place.
//
// swagger:model Holder
type Holder struct {
	// swagger:type Gadget
	Thing string `json:"thing"`
}
