// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"go/token"
	"go/types"

	"github.com/go-openapi/codescan/internal/builders/resolvers"
	"github.com/go-openapi/codescan/internal/ifaces"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	oaispec "github.com/go-openapi/spec"
)

// classifierAdditionalProperties applies a decl-level
// `swagger:additionalProperties <spec>` marker onto a top-level model schema.
//
// <spec> is `true` | `false` | a swagger:type-style spec (primitive / `[]T` /
// type-name → `$ref`). Semantics depend on the schema the Go type produced:
//
//   - struct → COMPLEMENT: keep the named properties, add additionalProperties;
//   - map → OVERRIDE the element-derived additionalProperties;
//   - a bare `$ref` (a map/wrapper type) → DEFINE a clean object schema (the
//     marker beats the Go type; a `$ref` cannot carry sibling keywords).
//
// additionalProperties is the lowest-priority annotation: if a prior rule has
// already resolved a non-object type (swagger:type on a non-object,
// swagger:strfmt, a special/known type), the marker is dropped with a
// diagnostic — it only ever rides on top of an object. See
// [§additional-properties](./README.md#additional-properties).
func (s *Builder) classifierAdditionalProperties(schema *oaispec.Schema, pos token.Position) {
	arg, ok := s.findAnnotationArg(s.Decl.Comments, grammar.AnnAdditionalProperties)
	if !ok {
		return
	}

	if len(schema.Type) > 0 && !schema.Type.Contains("object") {
		s.RecordDiagnostic(grammar.Warnf(pos, grammar.CodeShapeMismatch,
			"swagger:additionalProperties is only valid on an object schema; type is %q; ignored",
			schema.Type[0]))
		return
	}

	// A bare $ref (map/wrapper that resolved to a reference) is replaced by a
	// clean object: a $ref ignores sibling keywords, so additionalProperties
	// could not ride alongside it.
	if schema.Ref.String() != "" {
		schema.Ref = oaispec.Ref{}
	}
	schema.Typed("object", "")

	switch arg {
	case "true":
		schema.AdditionalProperties = &oaispec.SchemaOrBool{Allows: true}
	case "false":
		schema.AdditionalProperties = &oaispec.SchemaOrBool{Allows: false}
	default:
		apSchema := new(oaispec.Schema)
		if !s.resolveAdditionalPropertiesType(arg, apSchema, pos) {
			return // diagnostic already recorded
		}
		schema.AdditionalProperties = &oaispec.SchemaOrBool{Schema: apSchema}
	}
}

// resolveAdditionalPropertiesType resolves a swagger:type-style spec onto the
// additionalProperties value schema. Unlike swagger:type's resolveTypeOverride
// (which inlines named references), a type-name here resolves to a `$ref` — an
// additionalProperties value naturally references a model, matching how a
// `map[string]Model` field renders. Leading `[]` build array layers.
func (s *Builder) resolveAdditionalPropertiesType(arg string, target *oaispec.Schema, pos token.Position) bool {
	base, depth := stripArrayPrefixes(arg)

	var inner ifaces.SwaggerTypable = NewTypable(target, 0, s.skipExtensions)
	for range depth {
		inner.Typed("array", "")
		inner = inner.Items()
	}

	// Primitive / OAS-2 scalar / Go-builtin spelling — inline.
	if err := resolvers.SwaggerSchemaForType(base, inner); err == nil {
		return true
	}

	// Type-name reference → $ref (buildFromType on the named type, NOT its
	// Underlying, so buildNamedType emits the reference and registers it for
	// discovery).
	if s.refAdditionalPropertiesTypeName(base, inner) {
		return true
	}

	s.RecordDiagnostic(grammar.Warnf(pos, grammar.CodeUnsupportedType,
		"swagger:additionalProperties: unknown type %q", base))
	return false
}

// refAdditionalPropertiesTypeName resolves a type name declared in the
// builder's package onto target, producing a $ref for a model (or inlining a
// non-model leaf, exactly as a field of that type would). Returns false when
// there is no such declaration.
func (s *Builder) refAdditionalPropertiesTypeName(name string, tgt ifaces.SwaggerTypable) bool {
	if s.Decl == nil || s.Decl.Pkg == nil {
		return false
	}
	decl, ok := s.Ctx.FindDecl(s.Decl.Pkg.PkgPath, name)
	if !ok || decl == nil {
		return false
	}
	var t types.Type
	switch {
	case decl.Type != nil:
		t = decl.Type
	case decl.Alias != nil:
		t = decl.Alias
	default:
		return false
	}
	return s.buildFromType(t, tgt) == nil
}
