// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug1726 covers go-swagger#1726: markdown-style bullet lists
// (`* item`, `+ item`) must be identified as lists exactly like the YAML-style
// dash form (`- item`), everywhere the annotation language recognises a list.
// The lexer normalises `*`/`+` bullets to the canonical `- `, so both prose
// descriptions and value lists (consumes/produces/…) treat them identically.
//
// This fixture intentionally uses `*` and `+` bullets and is therefore NOT
// gofmt-clean: gofmt rewrites markdown bullets in doc comments to `-`. That is
// exactly the point — the scanner must normalise non-gofmt'd source the same
// way gofmt would, so the two forms agree. (fixtures/ is excluded from the
// gofmt formatters in .golangci.yml.)
package bug1726

// Widget uses the dash bullet form, the green guard rail that already worked.
//
// The widget supports:
//   - red
//   - green
//   - blue
//
// swagger:model
type Widget struct {
	// The colour.
	//
	// Allowed values:
	//   - red
	//   - green
	//   - blue
	Colour string `json:"colour"`
}

// Gadget uses asterisk bullets — the reporter's exact form. The leading `* `
// markers must survive as a list (normalised to `- `), not be stripped as
// godoc decoration into a run-on line.
//
// Features:
//   * fast
//   * cheap
//
// swagger:model
type Gadget struct {
	// kind of gadget
	Kind string `json:"kind"`
}

// Gizmo uses plus bullets, the third CommonMark bullet marker.
//
// Traits:
//   + light
//   + sturdy
//
// swagger:model
type Gizmo struct {
	// kind of gizmo
	Kind string `json:"kind"`
}

// ListWidgets exercises the value-list path (Produces/Consumes go through
// Property.AsList): markdown bullets there must be recognised as list items
// too, proving the relaxation propagates beyond prose descriptions.
//
// swagger:route GET /widgets widgets listWidgets
//
// List widgets.
//
// Produces:
//   * application/json
//   * application/xml
//
// Consumes:
//   + application/json
//
//	Responses:
//	  200: description: ok
func ListWidgets() {}
