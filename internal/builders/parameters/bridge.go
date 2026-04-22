// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parameters

import (
	"go/ast"
	"strings"

	"github.com/go-openapi/codescan/internal/builders/items"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/parsers/helpers"
	"github.com/go-openapi/codescan/internal/scanner/classify"
	oaispec "github.com/go-openapi/spec"
)

// paramItemsLevelTarget pairs a nesting depth (1-indexed, matching
// grammar.Property.ItemsDepth) with the *oaispec.Items to write
// items-level validations into. Parameter items form a chain via
// item.Items rather than schema's Items.Schema.Items.
type paramItemsLevelTarget struct {
	level int
	items *oaispec.Items
}

// collectParamItemsLevels mirrors items.ParseArrayTypes but collects
// (level, items) targets instead of building regex-based TagParsers.
// It is the grammar-path counterpart for parameter items dispatch.
//
// Starting level is 1 — `items.maximum:` has ItemsDepth=1 in the
// grammar lexer. Legacy v1 level=0 convention is re-indexed here.
func collectParamItemsLevels(expr ast.Expr, it *oaispec.Items, level int) []paramItemsLevelTarget {
	if it == nil {
		return nil
	}

	here := paramItemsLevelTarget{level: level, items: it}

	switch e := expr.(type) {
	case *ast.ArrayType:
		rest := collectParamItemsLevels(e.Elt, it.Items, level+1)
		out := make([]paramItemsLevelTarget, 0, 1+len(rest))
		return append(append(out, here), rest...)

	case *ast.Ident:
		rest := collectParamItemsLevels(expr, it.Items, level+1)
		if e.Obj == nil {
			out := make([]paramItemsLevelTarget, 0, 1+len(rest))
			return append(append(out, here), rest...)
		}
		return rest

	case *ast.StarExpr:
		return collectParamItemsLevels(e.X, it, level)

	case *ast.SelectorExpr:
		return []paramItemsLevelTarget{here}

	case *ast.StructType, *ast.InterfaceType, *ast.MapType:
		return nil

	default:
		return nil
	}
}

// applyBlockToField parses afld.Doc through the grammar parser and
// dispatches description, validations, required flag, extensions,
// and items-level validations into param. Replaces
// setupParamTaggers + SectionedParser under UseGrammarParser.
//
// Notes:
//   - `in:` is resolved upstream (via parsers.ParamLocation on the
//     raw comment group) before this bridge runs; grammar's lexer
//     also classifies it as TokenKeywordValue so it never reaches
//     the description accumulator. No dispatch here.
//   - Extension handling uses grammar's flat `block.Extensions()`
//     iterator, which captures `x-foo: value` lines inside an
//     `extensions:` block. YAML-fenced (`--- ... ---`) extension
//     blocks are not yet supported on the grammar path — no parity
//     fixture exercises them, and the follow-up commit that moves
//     YAML-body parsing through internal/parsers/yaml will also
//     plug this gap.
func (p *ParameterBuilder) applyBlockToField(afld *ast.Field, param *oaispec.Parameter) error {
	block := grammar.NewParser(p.decl.Pkg.Fset).Parse(afld.Doc)

	// Description: raw-line JoinDropLast for v1 parity (line-preserving
	// `"\n"` join), enum-desc extension suffix appended.
	param.Description = helpers.JoinDropLast(block.ProseLines())
	if enumDesc := helpers.GetEnumDesc(param.Extensions); enumDesc != "" {
		if param.Description != "" {
			param.Description += "\n"
		}
		param.Description += enumDesc
	}

	scheme := &param.SimpleSchema
	valid := paramValidations{param}

	for prop := range block.Properties() {
		if prop.ItemsDepth != 0 {
			continue
		}
		if err := dispatchParamKeyword(prop, param, valid, scheme); err != nil {
			return err
		}
	}

	for ext := range block.Extensions() {
		if !classify.IsAllowedExtension(ext.Name) {
			continue
		}
		param.AddExtension(ext.Name, ext.Value)
	}

	// items-level validation dispatch, mirroring items.ParseArrayTypes'
	// recursion. Only applies when the field type is written as an
	// array literal — named/alias array types opt out (parity).
	if arrayType, ok := afld.Type.(*ast.ArrayType); ok {
		for _, tgt := range collectParamItemsLevels(arrayType.Elt, param.Items, 1) {
			items.ApplyBlock(block, items.NewValidations(tgt.items), tgt.level)
		}
	}
	return nil
}

// dispatchParamKeyword routes a level-0 Property into paramValidations
// or the raw param target. Covers the same keyword surface as v1's
// baseInlineParamTaggers minus `in:` (upstream-resolved) and the
// Extensions block (handled via block.Extensions() by the caller).
func dispatchParamKeyword(p grammar.Property, param *oaispec.Parameter, valid paramValidations, scheme *oaispec.SimpleSchema) error {
	if dispatchNumericValidation(p, valid) {
		return nil
	}
	if dispatchIntegerValidation(p, valid) {
		return nil
	}
	handled, err := dispatchStringOrEnum(p, valid, scheme)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	dispatchParamFlags(p, param, valid)
	return nil
}

func dispatchNumericValidation(p grammar.Property, valid paramValidations) bool {
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

func dispatchIntegerValidation(p grammar.Property, valid paramValidations) bool {
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
// against the target's SimpleSchema. Parse errors from
// ParseValueFromSchema (e.g. `default: notanumber` on an int
// parameter) propagate so the run surfaces them, matching v1's
// error semantics.
func dispatchStringOrEnum(p grammar.Property, valid paramValidations, scheme *oaispec.SimpleSchema) (bool, error) {
	switch p.Keyword.Name {
	case "pattern":
		valid.SetPattern(p.Value)
	case "enum":
		valid.SetEnum(p.Value)
	case "default":
		v, err := helpers.ParseValueFromSchema(p.Value, scheme)
		if err != nil {
			return true, err
		}
		valid.SetDefault(v)
	case "example":
		v, err := helpers.ParseValueFromSchema(p.Value, scheme)
		if err != nil {
			return true, err
		}
		valid.SetExample(v)
	default:
		return false, nil
	}
	return true, nil
}

// dispatchParamFlags handles unique/required/collectionFormat — the
// remaining boolean and string-enum keywords that write to the
// parameter target.
func dispatchParamFlags(p grammar.Property, param *oaispec.Parameter, valid paramValidations) {
	switch p.Keyword.Name {
	case "unique":
		if p.Typed.Type == grammar.ValueBoolean {
			valid.SetUnique(p.Typed.Boolean)
		}
	case "required":
		if p.Typed.Type == grammar.ValueBoolean {
			param.Required = p.Typed.Boolean
		}
	case "collectionFormat":
		val := p.Typed.String
		if val == "" {
			val = strings.TrimSpace(p.Value)
		}
		if val != "" {
			valid.SetCollectionFormat(val)
		}
	}
}
