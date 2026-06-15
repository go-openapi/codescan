// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"go/token"
	"go/types"
	"strings"

	"github.com/go-openapi/codescan/internal/builders/resolvers"
	"github.com/go-openapi/codescan/internal/builders/validations"
	"github.com/go-openapi/codescan/internal/ifaces"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	oaispec "github.com/go-openapi/spec"
)

// resolveTypeOverride applies a `swagger:type` argument onto tgt, ALWAYS
// producing an inline schema (never a $ref). It is the single resolution
// point for the keyword consumed at every swagger:type site (the F3
// reconciliation — see .claude/plans/quirks-F-series-fix.md).
//
//   - ownType is the annotated field/decl's Go type, consumed by the
//     `inline` / `array` keywords (which expand that type in place). May be
//     nil when the site has no Go type to expand.
//   - pos drives diagnostics.
//
// Returns applied=true when the override was honoured (the caller
// short-circuits). applied=false means the caller should fall through to
// default Go-type resolution; a diagnostic explaining why has already been
// recorded (unknown type, `file`, or an invalid array element).
//
// Argument grammar (after stripping N leading `[]` → an N-deep array whose
// innermost items are the resolved base):
//
//   - keyword scalars `string`/`integer`/`number`/`boolean`/`object` and the
//     Go-builtin spellings (`int64`, `uint32`, `float64`, …) — via
//     resolvers.SwaggerSchemaForType;
//   - `inline` — expand ownType in place (no $ref; slice → array of inlined
//     items);
//   - `array` — deprecated alias of `inline` for collections (warns,
//     prefer `inline`);
//   - `file` — unsupported here (diagnostic; use swagger:file);
//   - any other token — a case-sensitive type-name reference, inlined from a
//     known definition; unknown → diagnostic.
func (s *Builder) resolveTypeOverride(arg string, tgt ifaces.SwaggerTypable, ownType types.Type, pos token.Position) (applied bool) {
	base, depth := stripArrayPrefixes(arg)
	if depth == 0 {
		return s.resolveTypeBase(base, tgt, ownType, pos, false)
	}

	// `[]T …` — build N array layers, then resolve the base into the
	// innermost items. The items form an inline schema like any other.
	tgt.Typed("array", "")
	items := tgt.Items()
	for range depth - 1 {
		items.Typed("array", "")
		items = items.Items()
	}
	return s.resolveTypeBase(base, items, ownType, pos, true)
}

// resolveTypeBase resolves a single (array-stripped) swagger:type base onto
// target. isElem reports whether target is an array-element position, where
// the own-type keywords (`inline`/`array`) and `file` are not meaningful.
func (s *Builder) resolveTypeBase(base string, target ifaces.SwaggerTypable, ownType types.Type, pos token.Position, isElem bool) (applied bool) {
	switch base {
	case keywordInline, keywordArray:
		if isElem {
			s.RecordDiagnostic(grammar.Warnf(pos, grammar.CodeUnsupportedType,
				"swagger:type: %q is not a valid array element type", base))
			return false
		}
		if base == keywordArray {
			s.RecordDiagnostic(grammar.Warnf(pos, grammar.CodeDeprecated,
				`swagger:type: "array" is deprecated, prefer "inline"`))
		}
		if ownType == nil {
			return false
		}
		return s.inlineGoType(ownType, target)
	case keywordFile:
		s.RecordDiagnostic(grammar.Warnf(pos, grammar.CodeUnsupportedType,
			`swagger:type: "file" is not supported here — use the swagger:file annotation instead`))
		return false
	}

	// Scalar / Go-builtin / canonical OAS-2 name.
	if err := resolvers.SwaggerSchemaForType(base, target); err == nil {
		return true
	}

	// Otherwise a type-name reference: inline a known definition in place.
	if s.inlineNamedTypeRef(base, target) {
		return true
	}

	s.RecordDiagnostic(grammar.Warnf(pos, grammar.CodeUnsupportedType,
		"swagger:type: unknown type %q", base))
	return false
}

// applyStrfmtFormat applies a swagger:strfmt format onto a schema whose type
// has already been fixed by swagger:type. The format rides as a supplementary
// hint ONLY when it is consistent with that type (string accepts any; integer
// / number accept the numeric width formats — see
// validations.IsFormatCompatible). An incompatible format is dropped with a
// shape-mismatch diagnostic rather than silently overriding the type. This is
// the swagger:type + swagger:strfmt precedence: type wins, format is advisory.
// It does NOT apply to the strfmt-alone path, where strfmt still forces
// {type: string, format: X} (go-swagger#1512).
func (s *Builder) applyStrfmtFormat(ps *oaispec.Schema, format string, pos token.Position) {
	var schemaType string
	if len(ps.Type) > 0 {
		schemaType = ps.Type[0]
	}
	ok, hint := validations.IsFormatCompatible(schemaType, format)
	if ok {
		ps.Format = format
		return
	}
	s.RecordDiagnostic(grammar.Warnf(pos, grammar.CodeShapeMismatch,
		"swagger:strfmt with swagger:type: %s; format ignored", hint))
}

// inlineGoType expands a Go type onto target as an inline schema, never a
// $ref: pointers are peeled and a named/alias type is reduced to its
// underlying shape (so buildFromType emits the structure rather than a
// reference). Returns true on success.
func (s *Builder) inlineGoType(t types.Type, target ifaces.SwaggerTypable) bool {
	base := t
	for {
		ptr, ok := base.(*types.Pointer)
		if !ok {
			break
		}
		base = ptr.Elem()
	}
	// Underlying() peels a Named/Alias to its structural type and is a
	// no-op for already-structural types — so buildFromType inlines
	// uniformly without taking the $ref branch in buildNamedType.
	return s.buildFromType(base.Underlying(), target) == nil
}

// inlineNamedTypeRef resolves name to a definition declared in the builder's
// current package and inlines its schema in place (no $ref). Returns false
// when the builder has no package context or no such definition exists.
func (s *Builder) inlineNamedTypeRef(name string, target ifaces.SwaggerTypable) bool {
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
	return s.buildFromType(t.Underlying(), target) == nil
}

// swagger:type keyword values that are not resolved as scalar/builtin type
// names (lowercase, case-sensitive — a capitalised spelling is a type-name
// reference instead, e.g. `Inline` vs the keyword `inline`).
const (
	keywordInline = "inline"
	keywordArray  = "array"
	keywordFile   = "file"
)

// stripArrayPrefixes counts leading `[]` prefixes on a swagger:type argument
// and returns the bare base plus the array depth. `[][]string` → ("string", 2),
// `int64` → ("int64", 0).
func stripArrayPrefixes(arg string) (base string, depth int) {
	base = strings.TrimSpace(arg)
	for strings.HasPrefix(base, "[]") {
		base = strings.TrimSpace(base[2:])
		depth++
	}
	return base, depth
}
