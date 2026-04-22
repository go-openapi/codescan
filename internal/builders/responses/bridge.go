// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package responses

import (
	"go/ast"

	"github.com/go-openapi/codescan/internal/builders/items"
	"github.com/go-openapi/codescan/internal/parsers"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	oaispec "github.com/go-openapi/spec"
)

// headerItemsLevelTarget pairs a 1-indexed nesting depth (matching
// grammar.Property.ItemsDepth) with an *oaispec.Items chain element
// for items-level validation dispatch.
type headerItemsLevelTarget struct {
	level int
	items *oaispec.Items
}

// collectHeaderItemsLevels mirrors items.ParseArrayTypes but collects
// (level, items) targets instead of building regex-based TagParsers.
// Header items form a chain via item.Items — same shape as parameters.
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

// applyBlockToDecl parses the top-level response doc under the
// grammar parser, writing the description to resp.Description via
// raw-line JoinDropLast (v1 parity). Does not dispatch any property
// keywords — the legacy top-level SectionedParser only accepts
// description, no taggers.
func (r *ResponseBuilder) applyBlockToDecl(resp *oaispec.Response) {
	block := grammar.NewParser(r.decl.Pkg.Fset).Parse(r.decl.Comments)
	resp.Description = parsers.JoinDropLast(block.ProseLines())
}

// applyBlockToHeader parses afld.Doc under the grammar parser and
// dispatches description, header validations, and items-level
// validations into ps. Replaces setupResponseHeaderTaggers +
// SectionedParser under UseGrammarParser.
//
// Notes:
//   - `in:` is resolved upstream (parsers.ParamLocation) before the
//     bridge runs; grammar's lexer also classifies it as
//     TokenKeywordValue so it never reaches the description
//     accumulator. No bridge dispatch.
//   - Headers have no `required:` — omitted from the flag dispatch.
//   - Extensions blocks are not currently supported on the header
//     path (no v1 tagger, no v2 dispatch); same status as parameters.
func (r *ResponseBuilder) applyBlockToHeader(afld *ast.Field, header *oaispec.Header) {
	block := grammar.NewParser(r.decl.Pkg.Fset).Parse(afld.Doc)

	header.Description = parsers.JoinDropLast(block.ProseLines())

	scheme := &header.SimpleSchema
	valid := headerValidations{header}

	for prop := range block.Properties() {
		if prop.ItemsDepth != 0 {
			continue
		}
		dispatchHeaderKeyword(prop, valid, scheme)
	}

	// items-level validation dispatch.
	if arrayType, ok := afld.Type.(*ast.ArrayType); ok {
		for _, tgt := range collectHeaderItemsLevels(arrayType.Elt, header.Items, 1) {
			items.ApplyBlock(block, items.NewValidations(tgt.items), tgt.level)
		}
	}
}

// dispatchHeaderKeyword routes a level-0 Property into
// headerValidations or, for scheme-aware default/example, through
// parsers.ParseValueFromSchema. Covers the v1 baseResponseHeaderTaggers
// surface minus `in:` (upstream-resolved).
func dispatchHeaderKeyword(p grammar.Property, valid headerValidations, scheme *oaispec.SimpleSchema) {
	if dispatchNumericValidation(p, valid) {
		return
	}
	if dispatchIntegerValidation(p, valid) {
		return
	}
	if dispatchStringOrEnum(p, valid, scheme) {
		return
	}
	dispatchHeaderFlags(p, valid)
}

func dispatchNumericValidation(p grammar.Property, valid headerValidations) bool {
	if p.Typed.Type != grammar.ValueNumber {
		return false
	}
	switch p.Keyword.Name {
	case "maximum":
		valid.SetMaximum(p.Typed.Number, p.Typed.Op == "<")
	case "minimum":
		valid.SetMinimum(p.Typed.Number, p.Typed.Op == ">")
	case "multipleOf":
		valid.SetMultipleOf(p.Typed.Number)
	default:
		return false
	}
	return true
}

func dispatchIntegerValidation(p grammar.Property, valid headerValidations) bool {
	if p.Typed.Type != grammar.ValueInteger {
		return false
	}
	switch p.Keyword.Name {
	case "minLength":
		valid.SetMinLength(p.Typed.Integer)
	case "maxLength":
		valid.SetMaxLength(p.Typed.Integer)
	case "minItems":
		valid.SetMinItems(p.Typed.Integer)
	case "maxItems":
		valid.SetMaxItems(p.Typed.Integer)
	default:
		return false
	}
	return true
}

// dispatchStringOrEnum handles pattern/enum/default/example —
// keywords whose value is consumed as a raw string or resolved
// against the target's SimpleSchema.
func dispatchStringOrEnum(p grammar.Property, valid headerValidations, scheme *oaispec.SimpleSchema) bool {
	switch p.Keyword.Name {
	case "pattern":
		valid.SetPattern(p.Value)
	case "enum":
		valid.SetEnum(p.Value)
	case "default":
		if v, err := parsers.ParseValueFromSchema(p.Value, scheme); err == nil {
			valid.SetDefault(v)
		}
	case "example":
		if v, err := parsers.ParseValueFromSchema(p.Value, scheme); err == nil {
			valid.SetExample(v)
		}
	default:
		return false
	}
	return true
}

// dispatchHeaderFlags handles unique/collectionFormat — the boolean
// and string-enum keywords that target the header. Headers have no
// required / readOnly / discriminator.
func dispatchHeaderFlags(p grammar.Property, valid headerValidations) {
	switch p.Keyword.Name {
	case "unique":
		if p.Typed.Type == grammar.ValueBoolean {
			valid.SetUnique(p.Typed.Boolean)
		}
	case "collectionFormat":
		if p.Typed.Type == grammar.ValueStringEnum {
			valid.SetCollectionFormat(p.Typed.String)
		}
	}
}
