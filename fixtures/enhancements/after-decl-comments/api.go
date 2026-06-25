// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package after_decl_comments witnesses the AfterDeclComments opt-in: swagger
// annotations live INSIDE a declaration (a struct's leading body comment) or
// INLINED as a trailing comment, so the godoc above the declaration stays
// clean. With the option off, those annotations are inert (nothing discovered).
//
// See .claude/plans/features/comment-source-filtering-design.md.
package after_decl_comments

// Widget is a widget. This godoc stays clean — no swagger machinery here.
type Widget struct {
	// Widget is exposed to API consumers.
	//
	// swagger:model widgetModel
	// maxProperties: 5

	Name string `json:"name"`
}

// Count is a plain count. Clean godoc above; annotation inlined below.
type Count int // swagger:model countType

// ListWidgets is an ordinary Go handler with a clean godoc.
func ListWidgets() {
	// swagger:route GET /widgets widgets listWidgets
	//
	// Lists the widgets.
	//
	// Responses:
	//   200: widgetModel
}
