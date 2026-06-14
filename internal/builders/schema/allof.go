// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"go/types"

	"github.com/go-openapi/codescan/internal/builders/resolvers"
	"github.com/go-openapi/codescan/internal/logger"
	"github.com/go-openapi/codescan/internal/scanner"
	oaispec "github.com/go-openapi/spec"
)

// scanEmbeddedFields walks st's anonymous fields and decides whether
// each embed contributes properties to the outer schema directly or
// becomes an `allOf` compound member.
//
// # Details
//
// See [§allof](./README.md#allof) — embed classification rules,
// `IsAllOfMember` semantics, and how the returned target ties to
// `buildFromStruct`'s second pass.
//
// Returns:
//   - target — the schema receiving properties; nil if no embed contributed,
//     `schema` itself for plain embeds, a fresh schema when allOf is in play.
//   - hasAllOf — true if at least one allOf member was appended.
func (s *Builder) scanEmbeddedFields(
	decl *scanner.EntityDecl, st *types.Struct, schema *oaispec.Schema, nameByJSON map[string]propOwner,
) (target *oaispec.Schema, hasAllOf bool, err error) {
	for fld := range st.Fields() {
		if !fld.Anonymous() {
			continue
		}

		afld := resolvers.FindASTField(decl.File, fld.Pos())
		if afld == nil {
			continue
		}

		fd := s.scanFieldDoc(afld)
		if fd.Ignored {
			continue
		}

		_, ignore, _, _, err := resolvers.ParseJSONTag(afld, fld.Name())
		if err != nil {
			return nil, false, err
		}
		if ignore {
			continue
		}

		if !fd.IsAllOfMember {
			// Plain embed (no `swagger:allOf` annotation) — inline the
			// embedded type's properties into the outer schema,
			// regardless of whether the embed is a direct named type
			// or an alias of one. The previous `!isAliased` guard here
			// silently promoted aliased embeds to allOf composition,
			// violating the documented contract that allOf is only
			// produced for explicitly-annotated embeds.
			if target == nil {
				target = schema
			}
			// A `required:` annotation on the embed applies to the
			// properties it promotes (go-swagger#2701). Thread it through
			// the recursion, restoring afterwards so sibling fields are
			// unaffected.
			saved := s.embedInherited
			s.embedInherited = s.ReadEmbedInheritance(afld.Doc, saved)
			err := s.buildEmbedded(fld.Type(), target, nameByJSON)
			s.embedInherited = saved
			if err != nil {
				return nil, false, err
			}
			continue
		}

		hasAllOf = true
		if target == nil {
			target = &oaispec.Schema{}
		}
		var newSch oaispec.Schema
		if err := s.buildAllOf(fld.Type(), &newSch); err != nil {
			return nil, false, err
		}

		if fd.AllOfClass != "" {
			schema.AddExtension("x-class", fd.AllOfClass)
		}
		schema.AllOf = append(schema.AllOf, newSch)
	}

	return target, hasAllOf, nil
}

// buildAllOf builds the schema for one allOf compound member. Peels
// pointers and routes named types and aliases to their dedicated
// helpers.
//
// # Details
//
// See [§allof](./README.md#allof) — the three-arm dispatch and why
// non-Named / non-Alias inputs are dropped silently with a logger
// warning rather than an error.
func (s *Builder) buildAllOf(tpe types.Type, schema *oaispec.Schema) error {
	switch ftpe := tpe.(type) {
	case *types.Pointer:
		return s.buildAllOf(ftpe.Elem(), schema)
	case *types.Named:
		return s.buildNamedAllOf(ftpe, schema)
	case *types.Alias:
		tgt := NewTypable(schema, 0, s.skipExtensions)
		return s.buildAlias(ftpe, tgt)
	default:
		logger.UnsupportedTypeKind("buildAllOf", ftpe)
		return nil
	}
}

// buildNamedAllOf resolves a named type appearing as an allOf member.
// Struct and interface underlyings share the same precedence shape:
// user-classifier first, then stdlib specials, then model lookup, then
// inline build.
//
// # Details
//
// See [§allof](./README.md#allof) — arm symmetry rationale and why
// `classifierAliasTargetStrfmt` is preferred over a comment-group-keyed
// variant.
func (s *Builder) buildNamedAllOf(ftpe *types.Named, schema *oaispec.Schema) error {
	tgt := NewTypable(schema, 0, s.skipExtensions)
	tio := ftpe.Obj()

	if s.classifierAliasTargetStrfmt(ftpe, tgt) {
		return nil
	}
	if applyStdlibSpecials(tio, tgt, s.skipExtensions) {
		return nil
	}

	decl, found := s.Ctx.GetModel(tio.Pkg().Path(), tio.Name())
	if !found {
		return fmt.Errorf("can't find source for named allOf member %s: %w", ftpe.String(), ErrSchema)
	}

	if decl.HasModelAnnotation() {
		return s.MakeRef(decl, tgt)
	}

	switch utpe := ftpe.Underlying().(type) {
	case *types.Struct:
		return s.buildFromStruct(decl, utpe, schema, make(map[string]propOwner))
	case *types.Interface:
		return s.buildFromInterface(decl, utpe, schema, make(map[string]propOwner))
	default:
		logger.UnsupportedTypeKind("buildNamedAllOf", utpe)
		return nil
	}
}
