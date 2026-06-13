// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug3138 reproduces go-swagger issue #3138 ("How To mark a field as
// deprecated?"). OAS2 spec.Schema has no native `deprecated` field, so model-
// and field-level deprecation is emitted as the `x-deprecated: true` vendor
// extension. Two triggers are supported, on both models and fields:
//
//   - the explicit `deprecated: true` annotation, and
//   - a godoc-style "Deprecated:" paragraph (pkgsite convention), so the mark
//     need not be repeated.
//
// Operation-level `deprecated: true` keeps using the native OAS2 field.
package bug3138

// Widget is a current model with a mix of deprecated and live fields.
//
// swagger:model Widget
type Widget struct {
	// The legacy name.
	//
	// deprecated: true
	OldName string `json:"oldName"`

	// LegacyID is the old identifier.
	//
	// Deprecated: use Name instead.
	LegacyID string `json:"legacyID"`

	// The current name.
	Name string `json:"name"`
}

// OldWidget is a model deprecated via the explicit annotation.
//
// deprecated: true
//
// swagger:model OldWidget
type OldWidget struct {
	// The name.
	Name string `json:"name"`
}

// RetiredWidget is a model deprecated via a godoc paragraph.
//
// Deprecated: use Widget instead.
//
// swagger:model RetiredWidget
type RetiredWidget struct {
	// The name.
	Name string `json:"name"`
}

// swagger:operation GET /widgets listWidgets
//
// ---
// deprecated: true
// responses:
//
//	"200":
//	  description: ok
func listWidgets() {}
