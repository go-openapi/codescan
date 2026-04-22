// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"go/ast"

	"github.com/go-openapi/codescan/internal/builders/items"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	oaispec "github.com/go-openapi/spec"
)

// itemsLevelTarget pairs a nesting depth (1-indexed, matching
// grammar.Property.ItemsDepth) with the schema to write items-level
// validations into.
type itemsLevelTarget struct {
	level  int
	schema *oaispec.Schema
}

// collectItemsLevels mirrors the walk performed by parseArrayTypes but
// collects (level, schema) targets instead of building regex-based
// TagParsers. It is the grammar-path counterpart of the legacy
// itemsTaggers recursion.
//
// The starting level is 1 — `items.maximum:` has ItemsDepth=1 under the
// grammar lexer. The legacy v1 level=0 convention is re-indexed here so
// the caller can pass the value directly to items.ApplyBlock.
func collectItemsLevels(expr ast.Expr, schemaItems *oaispec.SchemaOrArray, level int) []itemsLevelTarget {
	if schemaItems == nil || schemaItems.Schema == nil {
		return nil
	}

	here := itemsLevelTarget{level: level, schema: schemaItems.Schema}

	switch e := expr.(type) {
	case *ast.ArrayType:
		rest := collectItemsLevels(e.Elt, schemaItems.Schema.Items, level+1)
		out := make([]itemsLevelTarget, 0, 1+len(rest))
		return append(append(out, here), rest...)

	case *ast.Ident:
		rest := collectItemsLevels(expr, schemaItems.Schema.Items, level+1)
		if e.Obj == nil {
			out := make([]itemsLevelTarget, 0, 1+len(rest))
			return append(append(out, here), rest...)
		}
		return rest

	case *ast.StarExpr:
		return collectItemsLevels(e.X, schemaItems, level)

	case *ast.SelectorExpr:
		return []itemsLevelTarget{here}

	case *ast.StructType, *ast.InterfaceType, *ast.MapType:
		return nil

	default:
		return nil
	}
}

// applyItemsBridge parses fld.Doc through the grammar parser and
// dispatches every items-level validation property into the matching
// schema level via items.ApplyBlock. Called only when the scan's
// UseGrammarParser flag is set.
//
// Shadow semantics for P5.1b: the legacy regex itemsTaggers still
// run (they also claim items-prefix lines away from the description,
// so dropping them would cause `items.pattern:` bodies to leak into
// prose). Both paths write to the same ValidationBuilder target with
// the same values, so the second write is a no-op; this exercises the
// grammar path end-to-end without diverging from v1. Subsequent
// migration steps will drop the legacy path as schema-level keywords
// and description-claim behavior move to the grammar side.
//
// No-op when fld is nil, the field type is not an array literal, or
// the schema has no items sub-tree — matching the legacy guard in
// createParser that only invokes parseArrayTypes for *ast.ArrayType
// fields.
func (s *Builder) applyItemsBridge(fld *ast.Field, ps *oaispec.Schema) {
	if fld == nil || fld.Doc == nil || ps == nil {
		return
	}
	arrayType, ok := fld.Type.(*ast.ArrayType)
	if !ok {
		return
	}
	if !s.ctx.UseGrammarParser() {
		return
	}

	targets := collectItemsLevels(arrayType.Elt, ps.Items, 1)
	if len(targets) == 0 {
		return
	}

	block := grammar.NewParser(s.decl.Pkg.Fset).Parse(fld.Doc)
	for _, tgt := range targets {
		items.ApplyBlock(block, schemaValidations{tgt.schema}, tgt.level)
	}
}
