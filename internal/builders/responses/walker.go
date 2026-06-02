// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package responses

import (
	"go/ast"

	"github.com/go-openapi/codescan/internal/builders/handlers"
	oaispec "github.com/go-openapi/spec"
)

// headerItemsLevelTarget pairs a 1-indexed nesting depth (matching
// grammar.Property.ItemsDepth) with an *oaispec.Items target into
// which items-level validations at that depth must be written.
type headerItemsLevelTarget struct {
	level int
	items *oaispec.Items
}

// collectHeaderItemsLevels walks the AST array layers of a response
// header field and returns the (level, items) pairs reachable from
// the header's items chain. Mirrors items.ParseArrayTypes' shape on
// the grammar path; identical recursion shape to parameters'
// equivalent helper.
//
// Starting level is 1 — `items.maximum:` has ItemsDepth=1 in the
// grammar lexer. Named/aliased array types opt out (parity with
// v1's tagger pipeline).
func collectHeaderItemsLevels(expr ast.Expr, it *oaispec.Items, level int) []headerItemsLevelTarget {
	if it == nil {
		return nil
	}

	here := headerItemsLevelTarget{level: level, items: it}

	switch e := expr.(type) {
	case *ast.ArrayType:
		rest := collectHeaderItemsLevels(e.Elt, it.Items, level+1)
		out := make([]headerItemsLevelTarget, 0, 1+len(rest))
		return append(append(out, here), rest...)

	case *ast.Ident:
		rest := collectHeaderItemsLevels(expr, it.Items, level+1)
		if e.Obj == nil {
			out := make([]headerItemsLevelTarget, 0, 1+len(rest))
			return append(append(out, here), rest...)
		}
		return rest

	case *ast.StarExpr:
		return collectHeaderItemsLevels(e.X, it, level)

	case *ast.SelectorExpr:
		return []headerItemsLevelTarget{here}

	case *ast.StructType, *ast.InterfaceType, *ast.MapType:
		return nil

	default:
		return nil
	}
}

// applyBlockToDecl parses the top-level response doc through grammar
// and writes the description to resp.Description via the grammar
// parser's prose accumulator. Does not dispatch any property
// keywords — the v1 SectionedParser only accepted description at the
// top level, no taggers.
func (r *Builder) applyBlockToDecl(resp *oaispec.Response) {
	block := r.ParseBlock(r.Decl.Comments)
	resp.Description = block.Prose()
}

// applyBlockToHeader parses afld.Doc through grammar and dispatches
// description, header validations, items-level validations, and
// user-authored vendor extensions into ps.
//
// # Details
//
// See [§dispatch](./README.md#dispatch) — the three-phase Walker
// dispatch for headers, the omitted `required:` write, and how
// items-level validations chain.
func (r *Builder) applyBlockToHeader(afld *ast.Field, header *oaispec.Header) {
	block := r.ParseBlock(afld.Doc)

	header.Description = block.Prose()
	handlers.DispatchHeaderLevel0(block, header, r.RecordDiagnostic)

	if arrayType, ok := afld.Type.(*ast.ArrayType); ok {
		for _, tgt := range collectHeaderItemsLevels(arrayType.Elt, header.Items, 1) {
			handlers.DispatchItemsLevel(block, tgt.items, tgt.level, r.RecordDiagnostic)
		}
	}
}
