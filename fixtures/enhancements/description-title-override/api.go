// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package description_title_override witnesses the swagger:title /
// swagger:description override annotations (Q30 close-out): an explicit
// annotation replaces the godoc-derived title / description so the Go-facing
// doc comment can diverge from the API-facing spec text.
//
// Scope exercised here (P2): model schemas + struct fields. The overrides
// replace the prose-derived value; absence leaves the godoc untouched
// (regression guard); a bare swagger:description suppresses the godoc (empty
// value applied) and raises scan.empty-override.
//
// See .claude/plans/features/swagger-description-override-design.md.
package description_title_override

// Widget is the Go-facing widget doc, written for Go readers.
//
// It explains internal Go usage that should not leak into the API spec.
//
// swagger:model
// swagger:title A Public Widget
// swagger:description A widget exposed via the public API.
type Widget struct {
	// ID explains the Go field for Go readers.
	//
	// swagger:description The unique widget identifier.
	ID string `json:"id"`

	// Label is the Go-facing field doc. Fields carry no title by default;
	// the override is the only way a property gets one.
	//
	// swagger:title Display Label
	// swagger:description Human-readable label shown to API consumers.
	Label string `json:"label"`

	// Plain keeps its godoc description because it carries no override
	Plain string `json:"plain"`

	// Suppressed has a godoc that is suppressed by a bare swagger:description:
	// the empty value is applied (description omitted) and scan.empty-override
	// is raised.
	//
	// swagger:description
	Suppressed string `json:"suppressed"`

	// Gadget is a $ref field carrying title + description overrides. Title and
	// description are symmetric $ref siblings: they ride the same preservation
	// rule (kept under EmitRefSiblings / a forced compound, dropped to a bare
	// $ref under the default flags) — no title-specific compounding.
	//
	// swagger:title Gadget Ref
	// swagger:description The attached gadget, described for API consumers.
	Gadget Gadget `json:"gadget"`
}

// Gadget is a plain referenced model.
//
// swagger:model
type Gadget struct {
	Serial string `json:"serial"`
}
