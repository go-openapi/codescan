// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"go/ast"

	"github.com/go-openapi/codescan/internal/builders/handlers"
	"github.com/go-openapi/codescan/internal/builders/resolvers"
	"github.com/go-openapi/codescan/internal/builders/validations"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner/classify"
	oaispec "github.com/go-openapi/spec"
)

// applyBlockToDecl is the grammar entry point for a top-level model
// declaration. Parses the doc, short-circuits on swagger:ignore, writes
// title/description, then dispatches schema-level properties via the
// Walker.
//
// Returns true when the block's primary annotation is swagger:ignore;
// the caller short-circuits further building.
func (s *Builder) applyDeclCommentBlock(schema *oaispec.Schema) (skip bool) {
	block := s.ParseBlock(s.Decl.Comments)
	// `swagger:ignore` only short-circuits when it is the FIRST
	// annotation on the comment group. Fixture
	// fixtures/enhancements/top-level-kinds/IgnoredModel deliberately
	// places `swagger:model` first and `swagger:ignore` second to
	// pin this behaviour: the ignore is silently overridden because
	// only the source-order-first annotation drives the short-circuit.
	// ParseAll widens visibility for inferNames-style discovery
	// (which IS source-order independent) but the ignore check stays
	// narrow on purpose.
	if block.AnnotationKind() == grammar.AnnIgnore {
		return true
	}

	schema.Title = block.PreambleTitle()
	schema.Description = block.PreambleDescription()
	if enumDesc := resolvers.GetEnumDesc(schema.Extensions); enumDesc != "" {
		if schema.Description != "" {
			schema.Description += "\n"
		}
		schema.Description += enumDesc
	}

	handlers.DispatchSchemaLevel0(block, schema, schema, "", s.RecordDiagnostic, s.schemaOpts())

	return false
}

// applyBlockToField is the grammar entry point for a struct field /
// interface method doc. Parses, dispatches level-0 properties, and
// recurses into items levels. When the field is a $ref to a named
// type and field-level sibling keywords are present, rewrites ps
// into an allOf compound: `{allOf: [{$ref: X}, {sibling overrides}]}`
// — JSON-Schema-draft-4 semantics so the override is preserved
// without dropping siblings of the $ref.
func (s *Builder) applyBlockToField(afld *ast.Field, enclosing *oaispec.Schema, ps *oaispec.Schema, name string) {
	block := s.ParseBlock(afld.Doc)

	if ps.Ref.String() != "" {
		s.applyToRefField(block, enclosing, ps, name)
		return
	}

	ps.Description = block.Prose()
	if enumDesc := resolvers.GetEnumDesc(ps.Extensions); enumDesc != "" {
		if ps.Description != "" {
			ps.Description += "\n"
		}
		ps.Description += enumDesc
	}

	handlers.DispatchSchemaLevel0(block, enclosing, ps, name, s.RecordDiagnostic, s.schemaOpts())

	// Items-level dispatch — only when the field type is written as
	// an array literal. Named/alias array types opt out: their items
	// chain belongs to the referenced/aliased definition, not to the
	// referring field's block.
	if arrayType, ok := afld.Type.(*ast.ArrayType); ok {
		targets := flattenItemsTargets(arrayType.Elt, ps.Items)
		for depth, target := range targets {
			handlers.DispatchSchemaItemsLevel(block, target, depth+1, s.RecordDiagnostic, s.schemaOpts())
		}
	}
}

// schemaOpts packages the Builder's dispatch options into the value
// the handlers entry points consume.
func (s *Builder) schemaOpts() handlers.SchemaOptions {
	return handlers.SchemaOptions{SimpleSchemaMode: s.simpleSchema}
}

// applyToRefField rewrites a $ref'd field into an allOf compound when
// field-level overrides are present.
//
// # Details
//
// See [§ref-override](./README.md#ref-override) — JSON-Schema-draft-4
// shape, per-keyword landing rules, the DescWithRef toggle, and the
// description-only edge case.
func (s *Builder) applyToRefField(block grammar.Block, enclosing, ps *oaispec.Schema, name string) {
	originalRef := ps.Ref

	c := &refOverrideCollector{builder: s, enclosing: enclosing, name: name}
	c.valid = handlers.NewSchemaValidations(&c.override)

	block.Walk(grammar.Walker{
		FilterDepth: 0,
		Number:      c.onNumber,
		Integer:     c.onInteger,
		Bool:        c.onBool,
		String:      c.onString,
		Raw:         c.onRaw,
		Extension:   c.onExtension,
		Diagnostic:  s.RecordDiagnostic,
	})

	description := block.Prose()

	if !c.anyCollected() && description == "" {
		return
	}
	if !c.anyCollected() && !s.Ctx.DescWithRef() {
		return
	}

	// Lift x-* siblings onto the outer compound (see §ref-override).
	liftedExtensions := c.override.Extensions
	c.override.Extensions = nil

	allOf := []oaispec.Schema{
		{SchemaProps: oaispec.SchemaProps{Ref: originalRef}},
	}
	if c.collectedValidation {
		allOf = append(allOf, c.override)
	}
	*ps = oaispec.Schema{
		VendorExtensible: oaispec.VendorExtensible{Extensions: liftedExtensions},
		SchemaProps: oaispec.SchemaProps{
			Description: description,
			AllOf:       allOf,
		},
	}
}

// refOverrideCollector accumulates field-level overrides into a
// scratch schema for the allOf compound rewrite.
//
// # Details
//
// See [§ref-override](./README.md#ref-override) — collector role,
// the two flags (`collectedValidation`, `collectedExtension`) and
// the lift-onto-outer behaviour for vendor extensions.
type refOverrideCollector struct {
	builder             *Builder
	enclosing           *oaispec.Schema
	name                string
	override            oaispec.Schema
	valid               handlers.SchemaValidations
	collectedValidation bool
	collectedExtension  bool
}

func (c *refOverrideCollector) anyCollected() bool {
	return c.collectedValidation || c.collectedExtension
}

func (c *refOverrideCollector) markValidation() { c.collectedValidation = true }
func (c *refOverrideCollector) markExtension()  { c.collectedExtension = true }

func (c *refOverrideCollector) onNumber(p grammar.Property, val float64, exclusive bool) {
	if !p.IsTyped() {
		return
	}
	switch p.Keyword.Name {
	case grammar.KwMaximum:
		c.valid.SetMaximum(val, exclusive)
		c.markValidation()
	case grammar.KwMinimum:
		c.valid.SetMinimum(val, exclusive)
		c.markValidation()
	case grammar.KwMultipleOf:
		c.valid.SetMultipleOf(val)
		c.markValidation()
	}
}

func (c *refOverrideCollector) onInteger(p grammar.Property, val int64) {
	if !p.IsTyped() {
		return
	}
	switch p.Keyword.Name {
	case grammar.KwMinLength:
		c.valid.SetMinLength(val)
		c.markValidation()
	case grammar.KwMaxLength:
		c.valid.SetMaxLength(val)
		c.markValidation()
	case grammar.KwMinItems:
		c.valid.SetMinItems(val)
		c.markValidation()
	case grammar.KwMaxItems:
		c.valid.SetMaxItems(val)
		c.markValidation()
	case grammar.KwMinProperties:
		c.valid.SetMinProperties(val)
		c.markValidation()
	case grammar.KwMaxProperties:
		c.valid.SetMaxProperties(val)
		c.markValidation()
	}
}

func (c *refOverrideCollector) onBool(p grammar.Property, val bool) {
	if !p.IsTyped() {
		return
	}
	switch p.Keyword.Name {
	case grammar.KwRequired:
		if c.name != "" {
			handlers.SetRequired(c.enclosing, c.name, val)
		}
	case grammar.KwReadOnly:
		c.override.ReadOnly = val
		c.markValidation()
	case grammar.KwUnique:
		c.valid.SetUnique(val)
		c.markValidation()
	}
}

func (c *refOverrideCollector) onString(p grammar.Property, val string) {
	switch p.Keyword.Name {
	case grammar.KwPattern:
		handlers.ApplyPattern(p, c.valid, val, c.builder.RecordDiagnostic)
		c.markValidation()
	case grammar.KwPatternProperties:
		handlers.ApplyPatternProperties(p, c.valid, val, c.builder.RecordDiagnostic)
		c.markValidation()
	case grammar.KwDefault:
		if v, err := validations.ParseDefault(val, handlers.SchemaTypeOf(&c.override), c.override.Format); err == nil {
			c.valid.SetDefault(v)
			c.markValidation()
		}
	case grammar.KwExample:
		if v, err := validations.ParseDefault(val, handlers.SchemaTypeOf(&c.override), c.override.Format); err == nil {
			c.valid.SetExample(v)
			c.markValidation()
		}
	case grammar.KwEnum:
		c.valid.SetEnum(val)
		c.markValidation()
	}
}

// onExtension applies one YAML-typed Extension entry onto the
// refOverride's compound and marks the collector so the outer caller
// emits an allOf wrap. Allowed-extension filtering matches the
// schema-level handler; user-authored extensions are not gated by
// SkipExtensions — SkipExtensions targets scanner-derived vendor
// extensions (`x-go-*`), not author-written ones.
func (c *refOverrideCollector) onExtension(ext grammar.Extension) {
	if !classify.IsAllowedExtension(ext.Name) {
		return
	}
	c.override.AddExtension(ext.Name, ext.Value)
	c.markExtension()
}

func (c *refOverrideCollector) onRaw(p grammar.Property) {
	switch p.Keyword.Name {
	case grammar.KwDefault:
		if v, err := validations.ParseDefault(p.Value, handlers.SchemaTypeOf(&c.override), c.override.Format); err == nil {
			c.valid.SetDefault(v)
			c.markValidation()
		}
	case grammar.KwExample:
		if v, err := validations.ParseDefault(p.Value, handlers.SchemaTypeOf(&c.override), c.override.Format); err == nil {
			c.valid.SetExample(v)
			c.markValidation()
		}
	case grammar.KwEnum:
		c.valid.SetEnum(p.Value)
		c.markValidation()
	}
}

// flattenItemsTargets walks the array-element AST in parallel with
// the schema's items chain and returns a flat slice of property
// schemas, one per nesting depth, indexed by depth-1 (i.e. depth=1
// → out[0]).
func flattenItemsTargets(elt ast.Expr, schemaItems *oaispec.SchemaOrArray) []*oaispec.Schema {
	var out []*oaispec.Schema
	for schemaItems != nil && schemaItems.Schema != nil {
		out = append(out, schemaItems.Schema)
		switch e := elt.(type) {
		case *ast.ArrayType:
			elt = e.Elt
			schemaItems = schemaItems.Schema.Items
		case *ast.Ident:
			if e.Obj == nil {
				schemaItems = schemaItems.Schema.Items
				continue
			}
			return out
		case *ast.StarExpr:
			elt = e.X
		case *ast.SelectorExpr:
			return out
		case *ast.StructType, *ast.InterfaceType, *ast.MapType:
			out = out[:len(out)-1]
			return out
		default:
			return out
		}
	}
	return out
}
