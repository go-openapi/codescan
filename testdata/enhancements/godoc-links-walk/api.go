// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package godoclinkswalk

// Gadget is a small component.
//
// swagger:model
type Gadget struct {
	// Name labels this [Gadget].
	Name string `json:"name"`
}

// Widget holds a [Gadget].
//
// Each nested shape below puts a doc-link somewhere the document walk has to reach on its own:
// composition arms, array items, map values and pattern properties are all schemas of their own.
//
// This also links to the [Catalogue], which nothing references — so under PruneUnusedModels the
// link resolves to a definition that is no longer in the document.
//
// swagger:model
type Widget struct {
	// Parts lists every [Gadget] fitted.
	Parts []Gadget `json:"parts"`

	// Index maps a code to the [Gadget] it selects.
	Index map[string]Gadget `json:"index"`
}

// Catalogue keys vendor entries off a pattern, each one a [Gadget].
//
// swagger:model
// swagger:patternProperties "^x-": Gadget
type Catalogue struct {
	// Known is the one named property beside the patterned ones.
	Known string `json:"known"`
}

// Assembly composes a [Widget] with fields of its own, so the finished schema is an allOf whose
// sibling arm carries prose.
//
// swagger:model
type Assembly struct {
	// swagger:allOf
	Widget

	// Serial identifies this [Assembly] on the line.
	Serial string `json:"serial"`
}
