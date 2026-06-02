// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package wrapper_decl_type_override isolates Gap B' — the
// wrapper-type's top-level definition emits an empty schema even
// though the same wrapper carries `swagger:type` decoration that
// the classifier honours at field reference sites.
//
// No referencing struct is declared here: the wrapper itself is the
// only top-level model, so the golden makes the gap unambiguous.
//
// Expected (target shape): the top-level `BareWrapperObject`
// definition should be `{type: object}`; `BareWrapperArray` should
// be `{type: array, items: {integer, uint8}}`. Today both come out
// empty (`x-go-package` + description only).
package wrapper_decl_type_override

import "encoding/json"

// BareWrapperObject — named wrapper of json.RawMessage with
// `swagger:type object`. No field references it; the only schema
// the scanner emits for this package is the top-level definition.
//
// swagger:model BareWrapperObject
// swagger:type object
type BareWrapperObject json.RawMessage

// BareWrapperArray — named wrapper with `swagger:type array`.
// No field references.
//
// swagger:model BareWrapperArray
// swagger:type array
type BareWrapperArray json.RawMessage
