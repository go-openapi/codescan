// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/go-openapi/codescan/internal/builders/handlers"
	"github.com/go-openapi/codescan/internal/builders/resolvers"
	"github.com/go-openapi/codescan/internal/builders/validations"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/scanner/classify"
	oaispec "github.com/go-openapi/spec"
)

// recordValidationOrigins anchors each scalar validation keyword in block to its
// source comment line, so following e.g. a `maximum` node in the spec jumps to
// its `// maximum: 100` line rather than the struct field. Only when a base path
// was initiated (WithPath) and a sink is wired. The keyword→segment knowledge
// lives in the grammar ([grammar.PointerPath]); here we prepend the field base
// and any items depth (base + (/items)×ItemsDepth + /segment, mirroring where
// the value renders). Runs only on the non-$ref field path (a $ref field with
// siblings is rewritten to an allOf compound elsewhere, so its validations are
// not children of base — they resolve to the field anchor).
func (s *Builder) recordValidationOrigins(block grammar.Block) {
	if s.path == "" || !s.Ctx.OriginEnabled() {
		return
	}
	emit := func(p grammar.Property) {
		ctx := grammar.CtxSchema
		if p.ItemsDepth > 0 {
			ctx = grammar.CtxItems
		}
		segs, ok := grammar.PointerPath(p.Keyword, ctx)
		if !ok {
			return
		}
		ptr := s.path + strings.Repeat(scanner.JSONPointer("items"), p.ItemsDepth) + scanner.JSONPointer(segs...)
		s.Ctx.RecordOrigin(ptr, p.Pos)
	}
	block.Walk(grammar.Walker{
		FilterDepth: grammar.AllDepths,
		Number:      func(p grammar.Property, _ float64, _ bool) { emit(p) },
		Integer:     func(p grammar.Property, _ int64) { emit(p) },
		Bool:        func(p grammar.Property, _ bool) { emit(p) },
		String:      func(p grammar.Property, _ string) { emit(p) },
		Raw:         func(p grammar.Property) { emit(p) },
	})
}

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
	schema.Description = resolvers.AppendEnumDesc(schema.Description, schema.Extensions, s.Ctx.SkipEnumDescriptions())

	// `deprecated: true` or a godoc-style "Deprecated:" paragraph marks the
	// model deprecated (go-swagger#3138). The grammar block unifies both
	// triggers; OAS2 has no native schema `deprecated`, so emit x-deprecated.
	if block.IsDeprecated() {
		resolvers.MarkDeprecated(schema)
	}

	handlers.DispatchSchemaLevel0(block, schema, schema, "", s.RecordDiagnostic, s.schemaOpts())

	// Cross-ref linkage: anchor decl-level validation keywords to their lines.
	s.recordValidationOrigins(block)

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
	ps.Description = resolvers.AppendEnumDesc(ps.Description, ps.Extensions, s.Ctx.SkipEnumDescriptions())

	// `deprecated: true` or a godoc-style "Deprecated:" paragraph marks the
	// field deprecated (go-swagger#3138) — see the model-level note above.
	if block.IsDeprecated() {
		resolvers.MarkDeprecated(ps)
	}

	handlers.DispatchSchemaLevel0(block, enclosing, ps, name, s.RecordDiagnostic, s.schemaOpts())

	// additionalProperties: <spec> field keyword. Applied after the type-derived
	// dispatch so it complements an inline object, overrides a map's element
	// schema, or warn-drops on a non-object — the same precedence as the
	// type-level marker. ($ref'd fields are handled in applyToRefField via an
	// allOf sibling, above.)
	if apSpec, ok := block.GetString(grammar.KwAdditionalProperties); ok {
		s.applyAdditionalPropertiesSpec(ps, strings.TrimSpace(apSpec), s.Ctx.PosOf(afld.Pos()))
	}

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

	// Cross-ref linkage: anchor each validation keyword to its comment line.
	s.recordValidationOrigins(block)
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
		return // bare {$ref}: nothing to attach
	}

	// SkipAllOfCompounding: never emit an allOf compound. Validations and
	// externalDocs can only ride a compound, so they are dropped;
	// description and extensions are dropped too UNLESS EmitRefSiblings
	// keeps them as direct $ref siblings. `required` already landed on the
	// enclosing schema during the Walk (a parent-side concern, not a $ref
	// sibling) and is unaffected.
	if s.Ctx.SkipAllOfCompounding() {
		s.applyRefSiblingDrop(c, ps, description, name, block.Pos())
		return
	}

	// EmitRefSiblings: when nothing forces a compound (no validations, no
	// externalDocs), description and extensions ride directly beside the
	// $ref rather than in a single-arm allOf wrap. A forced compound falls
	// through to the wrap path below, where they ride the outer compound.
	forcedCompound := c.collectedValidation || c.collectedExternalDoc
	if s.Ctx.EmitRefSiblings() && !forcedCompound {
		ps.Description = description
		for k, v := range c.override.Extensions {
			ps.AddExtension(k, v)
		}
		return
	}

	if !c.anyCollected() && !s.Ctx.DescWithRef() {
		return // description-only, not preserved → bare {$ref}
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
		// externalDocs is an annotation sibling of the $ref, like
		// description and x-* — it lifts onto the outer compound
		// rather than into the allOf override (go-swagger#2655).
		SwaggerSchemaProps: oaispec.SwaggerSchemaProps{
			ExternalDocs: c.externalDocs,
		},
	}
}

// applyRefSiblingDrop handles the SkipAllOfCompounding case: no allOf
// compound is produced, so the field keeps its bare {$ref}. Extensions
// survive as direct siblings when EmitRefSiblings is set; description
// likewise. Everything else (validations, externalDocs) — and, without
// EmitRefSiblings, description / extensions too — is dropped, each with
// one CodeDroppedRefSibling diagnostic so the loss is never silent.
func (s *Builder) applyRefSiblingDrop(c *refOverrideCollector, ps *oaispec.Schema, description, name string, blockPos token.Position) {
	keepSiblings := s.Ctx.EmitRefSiblings()

	for _, d := range c.collected {
		if keepSiblings && d.kind == siblingExtension {
			ps.AddExtension(d.keyword, c.override.Extensions[d.keyword])
			continue
		}
		s.RecordDiagnostic(grammar.Warnf(d.pos, grammar.CodeDroppedRefSibling,
			"field %q: %q dropped — not representable on a bare $ref (SkipAllOfCompounding)",
			name, d.keyword))
	}

	if description == "" {
		return
	}
	if keepSiblings {
		ps.Description = description
		return
	}
	s.RecordDiagnostic(grammar.Warnf(blockPos, grammar.CodeDroppedRefSibling,
		"field %q: description dropped — not representable on a bare $ref (SkipAllOfCompounding)",
		name))
}

// refOverrideCollector accumulates field-level overrides into a
// scratch schema for the allOf compound rewrite.
//
// # Details
//
// See [§ref-override](./README.md#ref-override) — collector role,
// the flags (`collectedValidation`, `collectedExtension`,
// `collectedExternalDoc`) and the lift-onto-outer behaviour for
// vendor extensions and externalDocs.
type refOverrideCollector struct {
	builder              *Builder
	enclosing            *oaispec.Schema
	name                 string
	override             oaispec.Schema
	valid                handlers.SchemaValidations
	externalDocs         *oaispec.ExternalDocumentation
	collectedValidation  bool
	collectedExtension   bool
	collectedExternalDoc bool
	// collected records each collected sibling (keyword + source
	// position + class) so applyToRefField can decide, per category,
	// what to drop under SkipAllOfCompounding and raise a per-keyword
	// diagnostic. See [§ref-override].
	collected []collectedSibling
}

// siblingKind classifies a $ref sibling by how it can be emitted.
// Extensions can ride directly beside a $ref (EmitRefSiblings);
// validations and externalDocs can only ride an allOf compound.
type siblingKind int

const (
	siblingValidation siblingKind = iota
	siblingExtension
	siblingExternalDoc
)

// collectedSibling names one $ref-sibling keyword, where it was written,
// and its class — for the drop diagnostics and category-aware handling.
type collectedSibling struct {
	keyword string
	pos     token.Position
	kind    siblingKind
}

func (c *refOverrideCollector) anyCollected() bool {
	return c.collectedValidation || c.collectedExtension || c.collectedExternalDoc
}

func (c *refOverrideCollector) markValidation(p grammar.Property) {
	c.collectedValidation = true
	c.collected = append(c.collected, collectedSibling{keyword: p.Keyword.Name, pos: p.Pos, kind: siblingValidation})
}

func (c *refOverrideCollector) markExtension(ext grammar.Extension) {
	c.collectedExtension = true
	c.collected = append(c.collected, collectedSibling{keyword: ext.Name, pos: ext.Pos, kind: siblingExtension})
}

func (c *refOverrideCollector) onNumber(p grammar.Property, val float64, exclusive bool) {
	if !p.IsTyped() {
		return
	}
	switch p.Keyword.Name {
	case grammar.KwMaximum:
		c.valid.SetMaximum(val, exclusive)
		c.markValidation(p)
	case grammar.KwMinimum:
		c.valid.SetMinimum(val, exclusive)
		c.markValidation(p)
	case grammar.KwMultipleOf:
		c.valid.SetMultipleOf(val)
		c.markValidation(p)
	}
}

func (c *refOverrideCollector) onInteger(p grammar.Property, val int64) {
	if !p.IsTyped() {
		return
	}
	switch p.Keyword.Name {
	case grammar.KwMinLength:
		c.valid.SetMinLength(val)
		c.markValidation(p)
	case grammar.KwMaxLength:
		c.valid.SetMaxLength(val)
		c.markValidation(p)
	case grammar.KwMinItems:
		c.valid.SetMinItems(val)
		c.markValidation(p)
	case grammar.KwMaxItems:
		c.valid.SetMaxItems(val)
		c.markValidation(p)
	case grammar.KwMinProperties:
		c.valid.SetMinProperties(val)
		c.markValidation(p)
	case grammar.KwMaxProperties:
		c.valid.SetMaxProperties(val)
		c.markValidation(p)
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
		c.markValidation(p)
	case grammar.KwUnique:
		c.valid.SetUnique(val)
		c.markValidation(p)
	}
}

func (c *refOverrideCollector) onString(p grammar.Property, val string) {
	switch p.Keyword.Name {
	case grammar.KwPattern:
		handlers.ApplyPattern(p, c.valid, val, c.builder.RecordDiagnostic)
		c.markValidation(p)
	case grammar.KwPatternProperties:
		handlers.ApplyPatternProperties(p, c.valid, val, c.builder.RecordDiagnostic)
		c.markValidation(p)
	case grammar.KwAdditionalProperties:
		// On a $ref'd field, additionalProperties rides as an allOf sibling
		// (`{allOf: [{$ref}, {additionalProperties: …}]}`) so the reference is
		// preserved — JSON-Schema-draft-4 semantics, like the other siblings.
		if sob, ok := c.builder.resolveAdditionalPropertiesValue(strings.TrimSpace(val), p.Pos); ok {
			c.override.AdditionalProperties = sob
			c.markValidation(p)
		}
	case grammar.KwDefault:
		// The $ref override arm carries no Type of its own, so a JSON
		// object/array literal is coerced structurally here rather than
		// type-driven via ParseDefault (quirk G3).
		c.valid.SetDefault(validations.CoerceJSONOrString(val))
		c.markValidation(p)
	case grammar.KwExample:
		c.valid.SetExample(validations.CoerceJSONOrString(val))
		c.markValidation(p)
	case grammar.KwEnum:
		c.valid.SetEnum(val)
		c.markValidation(p)
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
	c.markExtension(ext)
}

func (c *refOverrideCollector) onRaw(p grammar.Property) {
	switch p.Keyword.Name {
	case grammar.KwDefault:
		// See onString: type-unknown override arm → JSON-literal coercion.
		c.valid.SetDefault(validations.CoerceJSONOrString(p.Value))
		c.markValidation(p)
	case grammar.KwExample:
		c.valid.SetExample(validations.CoerceJSONOrString(p.Value))
		c.markValidation(p)
	case grammar.KwEnum:
		c.valid.SetEnum(p.Value)
		c.markValidation(p)
	case grammar.KwExternalDocs:
		// externalDocs on a $ref'd field lifts onto the outer allOf
		// compound (see applyToRefField). A non-ref field handles it
		// via handlers.schemaRawHandler instead (go-swagger#2655).
		ed, err := handlers.ParseExternalDocs(p.Body)
		if err != nil {
			c.builder.RecordDiagnostic(grammar.Warnf(p.Pos, grammar.CodeInvalidAnnotation, "externalDocs: %v", err))
			return
		}
		if ed != nil {
			c.externalDocs = ed
			c.collectedExternalDoc = true
			c.collected = append(c.collected, collectedSibling{keyword: p.Keyword.Name, pos: p.Pos, kind: siblingExternalDoc})
		}
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
