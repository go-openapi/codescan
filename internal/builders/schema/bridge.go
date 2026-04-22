// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"go/ast"
	"strings"

	"github.com/go-openapi/loads/fmts"
	yaml "go.yaml.in/yaml/v3"

	"github.com/go-openapi/codescan/internal/builders/items"
	"github.com/go-openapi/codescan/internal/ifaces"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/parsers/helpers"
	"github.com/go-openapi/codescan/internal/scanner/classify"
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

// schemaBlockTargets bundles the write surfaces used by
// applySchemaBlock. The same schema pointer may fill both fields when
// there is no distinct enclosing type (e.g., the top-level model
// declaration doc).
type schemaBlockTargets struct {
	// enclosing is the schema that owns the current property — writes
	// for `required:` / `discriminator:` target its Required slice /
	// Discriminator field.
	enclosing *oaispec.Schema
	// ps is the schema describing the current property itself — writes
	// for numeric/string validations, readOnly, extensions, etc.
	ps *oaispec.Schema
	// name is the property's JSON name inside enclosing — used to
	// key required/discriminator. Empty when the bridge applies to a
	// top-level declaration (no enclosing index).
	name string
}

// applySchemaBlock dispatches every level-0 Property in b into the
// appropriate write target. It is the grammar-side replacement for the
// union of schemaTaggers + enum/required/readOnly/discriminator/
// YAMLExtensionsBlock parsers.
//
// Level-0 properties (ItemsDepth == 0) go through this dispatch;
// level-≥1 properties are handled by items.ApplyBlock invoked per
// nesting level by the caller.
//
// Keyword coverage and semantics mirror schemaTaggers in
// internal/builders/schema/taggers.go. See
// .claude/plans/p5.1b-schema-walkthrough.md §3 for the full mapping
// table.
func applySchemaBlock(b grammar.Block, t schemaBlockTargets) {
	scheme := schemeFromPS(t.ps)
	valid := schemaValidations{t.ps}

	for p := range b.Properties() {
		if p.ItemsDepth != 0 {
			continue
		}
		if p.Keyword.Name == "extensions" || p.Keyword.Name == "YAMLExtensionsBlock" {
			// Delegate the body to v1's YAML-aware extension parser
			// so nested/typed values (bool, list, map) are
			// recognised — block.Extensions()'s flat iterator
			// stores values as strings only.
			applyExtensionsBody(t.ps, p.Body)
			continue
		}
		dispatchSchemaKeyword(p, t, valid, scheme)
	}
}

// applyExtensionsBody feeds the grammar-captured extension body
// lines through a YAML → JSON pipeline so nested / typed values
// (bool, number, list, map) land on ps.Extensions with their
// semantic types — parity with the legacy schemaVendorExtensibleSetter
// path. Unknown x-* names (rejected by IsAllowedExtension) are
// silently dropped, matching the legacy reject-with-error behaviour
// sufficiently for parity (errors on extension names are rare and
// always user-authored).
func applyExtensionsBody(ps *oaispec.Schema, body []string) {
	if len(body) == 0 || (len(body) == 1 && body[0] == "") {
		return
	}
	yamlContent := strings.Join(body, "\n")
	var yamlValue any
	if err := yaml.Unmarshal([]byte(yamlContent), &yamlValue); err != nil {
		return
	}
	jsonValue, err := fmts.YAMLToJSON(yamlValue)
	if err != nil {
		return
	}
	var data oaispec.Extensions
	if err := json.Unmarshal(jsonValue, &data); err != nil {
		return
	}
	for k, v := range data {
		if !classify.IsAllowedExtension(k) {
			continue
		}
		ps.AddExtension(k, v)
	}
}

func dispatchSchemaKeyword(p grammar.Property, t schemaBlockTargets, valid schemaValidations, scheme *oaispec.SimpleSchema) {
	if dispatchNumericValidation(p, valid) {
		return
	}
	if dispatchIntegerValidation(p, valid) {
		return
	}
	if dispatchStringOrEnum(p, valid, scheme) {
		return
	}
	dispatchFlagValidation(p, t, valid)
	// Unrecognized keywords fall through silently. The grammar parser
	// already emitted a context-validity diagnostic when the keyword
	// was not legal here. `in:` is a match-only directive (see legacy
	// NewMatchIn); grammar's line classification already excludes it
	// from description prose, no write needed.
}

func dispatchNumericValidation(p grammar.Property, valid schemaValidations) bool {
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

func dispatchIntegerValidation(p grammar.Property, valid schemaValidations) bool {
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

// dispatchStringOrEnum handles pattern/enum/default/example — the
// four keywords whose value is consumed as a raw string or resolved
// against the target scheme rather than a pre-typed primitive.
func dispatchStringOrEnum(p grammar.Property, valid schemaValidations, scheme *oaispec.SimpleSchema) bool {
	switch p.Keyword.Name {
	case "pattern":
		valid.SetPattern(p.Value)
	case "enum":
		// Parity-first: route through v1's ParseEnum. Switching to
		// internal/parsers/enum.Parse is a post-migration quirk-fix
		// (see .claude/plans/workshops/w2-enum.md §2.6).
		valid.SetEnum(p.Value)
	case "default":
		if v, err := helpers.ParseValueFromSchema(p.Value, scheme); err == nil {
			valid.SetDefault(v)
		}
	case "example":
		if v, err := helpers.ParseValueFromSchema(p.Value, scheme); err == nil {
			valid.SetExample(v)
		}
	default:
		return false
	}
	return true
}

// dispatchFlagValidation handles unique/required/readOnly/discriminator
// — boolean-typed keywords. required/discriminator key on the property
// name and write to the enclosing schema; unique/readOnly write to ps.
func dispatchFlagValidation(p grammar.Property, t schemaBlockTargets, valid schemaValidations) {
	if p.Typed.Type != grammar.ValueBoolean {
		return
	}
	switch p.Keyword.Name {
	case "unique":
		valid.SetUnique(p.Typed.Boolean)
	case "readOnly":
		t.ps.ReadOnly = p.Typed.Boolean
	case "required":
		if t.name != "" {
			setRequired(t.enclosing, t.name, p.Typed.Boolean)
		}
	case "discriminator":
		if t.name != "" {
			setDiscriminator(t.enclosing, t.name, p.Typed.Boolean)
		}
	}
}

// setRequired adds or removes name from the enclosing schema's
// Required slice. Mirrors parsers.SetRequiredSchema.
func setRequired(enclosing *oaispec.Schema, name string, required bool) {
	if enclosing == nil {
		return
	}
	midx := -1
	for i, nm := range enclosing.Required {
		if nm == name {
			midx = i
			break
		}
	}
	if required {
		if midx < 0 {
			enclosing.Required = append(enclosing.Required, name)
		}
		return
	}
	if midx >= 0 {
		enclosing.Required = append(enclosing.Required[:midx], enclosing.Required[midx+1:]...)
	}
}

// setDiscriminator writes name to enclosing.Discriminator when
// required=true, or clears it when required=false and the current
// value matches. Mirrors parsers.SetDiscriminator.
func setDiscriminator(enclosing *oaispec.Schema, name string, required bool) {
	if enclosing == nil {
		return
	}
	if required {
		enclosing.Discriminator = name
		return
	}
	if enclosing.Discriminator == name {
		enclosing.Discriminator = ""
	}
}

// schemeFromPS builds the SimpleSchema that legacy NewSetDefault /
// NewSetExample take at construction time, derived from the already-
// populated ps.Type written by buildFromType before the comment
// dispatch.
//
// Quirk-preserving parity: v1 constructs the scheme with
// `Type: string(ps.Type.MarshalJSON())` and deliberately leaves
// Format empty. SimpleSchema.TypeName() returns Format when set,
// which would flip dispatch from the "number"/"integer" cases to
// format-specific strings ("float", "int32") that ParseValueFromSchema
// doesn't recognize. The pre-migration quirk is to ignore Format
// here; mirroring it keeps legacy fixtures (e.g., a float32 field
// with `default: 1.5`) parsing correctly.
//
// The MarshalJSON-derived Type contains JSON quote characters around
// the string, which ParseValueFromSchema strips via
// strings.Trim(..., `"`). Our version passes the unquoted token
// directly; the strip is a no-op in that path.
func schemeFromPS(ps *oaispec.Schema) *oaispec.SimpleSchema {
	if ps == nil {
		return nil
	}
	var typ string
	if len(ps.Type) > 0 {
		typ = ps.Type[0]
	}
	return &oaispec.SimpleSchema{Type: typ}
}

// --- orchestrators invoked from schema.go call sites ----------------

// applyBlockToField is the grammar-path counterpart of
// `sp := s.createParser(...); sp.Parse(afld.Doc)` for a struct field
// or interface method. It parses the doc once, writes the description
// from the grammar's raw prose lines (v1 parity: line-preserving
// "\n" join, not paragraph-joined), dispatches schema-level
// properties, and recurses into items levels.
//
// When ps.Ref is set and DescWithRef is false, mirrors legacy
// refSchemaTaggers by only dispatching `required:`.
func (s *Builder) applyBlockToField(afld *ast.Field, enclosing *oaispec.Schema, ps *oaispec.Schema, name string) {
	block := grammar.NewParser(s.decl.Pkg.Fset).Parse(afld.Doc)

	// $ref-mode: only `required:` applies, matching refSchemaTaggers.
	if ps.Ref.String() != "" && !s.ctx.DescWithRef() {
		for p := range block.Properties() {
			if p.Keyword.Name == "required" && p.ItemsDepth == 0 && p.Typed.Type == grammar.ValueBoolean {
				setRequired(enclosing, name, p.Typed.Boolean)
			}
		}
		return
	}

	// Field-level calls have no WithSetTitle callback in v1 — the
	// entire prose header is the description. Legacy output is
	// JoinDropLast("\n", header); enum-desc extension suffix is
	// appended last.
	ps.Description = helpers.JoinDropLast(block.ProseLines())
	if enumDesc := helpers.GetEnumDesc(ps.Extensions); enumDesc != "" {
		if ps.Description != "" {
			ps.Description += "\n"
		}
		ps.Description += enumDesc
	}

	applySchemaBlock(block, schemaBlockTargets{
		enclosing: enclosing,
		ps:        ps,
		name:      name,
	})

	// items-level validation dispatch, mirroring parseArrayTypes'
	// recursion. Only applies when the field type is written as an
	// array literal — named/alias array types opt out (parity).
	if arrayType, ok := afld.Type.(*ast.ArrayType); ok {
		for _, tgt := range collectItemsLevels(arrayType.Elt, ps.Items, 1) {
			items.ApplyBlock(block, schemaValidations{tgt.schema}, tgt.level)
		}
	}
}

// applyBlockToDecl is the grammar-path counterpart of the buildFromDecl
// SectionedParser call. Drives title/description and schema-level
// dispatch for a top-level model declaration doc. Preserves v1's
// title-vs-description split heuristics (first blank line splits, or
// punctuation/markdown-heading on the first line, otherwise all prose
// is description).
//
// Returns true when the block's primary annotation is swagger:ignore
// — the caller short-circuits further building, matching legacy
// sp.Ignored() semantics (first-annotation-wins: swagger:ignore must
// appear before any other swagger:* line to take effect).
//
// required/discriminator writes are no-ops at the declaration level
// because applySchemaBlock requires a non-empty property name.
func (s *Builder) applyBlockToDecl(schema *oaispec.Schema) (ignored bool) {
	block := grammar.NewParser(s.decl.Pkg.Fset).Parse(s.decl.Comments)

	if block.AnnotationKind() == grammar.AnnIgnore {
		return true
	}

	title, desc := helpers.CollectScannerTitleDescription(block.ProseLines())
	schema.Title = helpers.JoinDropLast(title)
	schema.Description = helpers.JoinDropLast(desc)
	if enumDesc := helpers.GetEnumDesc(schema.Extensions); enumDesc != "" {
		if schema.Description != "" {
			schema.Description += "\n"
		}
		schema.Description += enumDesc
	}

	applySchemaBlock(block, schemaBlockTargets{
		enclosing: schema,
		ps:        schema,
		name:      "", // no property index at the declaration level
	})
	return false
}

// Compile-time assertion: schemaValidations satisfies
// ifaces.ValidationBuilder, the target type used by both the schema
// bridge and the items bridge.
var _ ifaces.ValidationBuilder = schemaValidations{}
