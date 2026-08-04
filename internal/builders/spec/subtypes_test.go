// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"sort"
	"testing"

	"github.com/go-openapi/codescan/internal/scanner"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const (
	subtypesFixtureRoot = "github.com/go-openapi/codescan/fixtures/enhancements/discriminated-subtypes"
	teslaCarIdentity    = subtypesFixtureRoot + "/base.TeslaCar"
	plainBaseIdentity   = subtypesFixtureRoot + "/base.PlainBase"
	vehicleIdentity     = subtypesFixtureRoot + "/edges.Vehicle"
	vehicleAliasID      = subtypesFixtureRoot + "/edges.VehicleAlias"
)

// newSubtypesBuilder loads the discriminated-subtypes fixture and returns a Builder over it.
//
// ScanModels is off on purpose: the reverse index is fed by the model index, which classification populates either way
// — that independence is what makes the no-`-m` pull possible at all.
func newSubtypesBuilder(t *testing.T) *Builder {
	t.Helper()
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages: []string{"./enhancements/discriminated-subtypes/..."},
		WorkDir:  "../../../fixtures",
	})
	require.NoError(t, err)

	return NewBuilder(nil, ctx, false)
}

// declKeys is the sorted definition-key view of an index entry.
func declKeys(decls []*scanner.EntityDecl) []string {
	out := make([]string, 0, len(decls))
	for _, d := range decls {
		out = append(out, d.DefKey())
	}
	sort.Strings(out)

	return out
}

// findModelDecl returns the model declaration whose definition key ends in /name.
func findModelDecl(t *testing.T, b *Builder, name string) *scanner.EntityDecl {
	t.Helper()
	for _, decl := range b.ctx.Models() {
		if leafName(decl.DefKey()) == name {
			return decl
		}
	}
	require.FailNowf(t, "model not found", "no swagger:model declaration named %q in the fixture", name)

	return nil
}

// TestSubtypeIndex locks the reverse `swagger:allOf` index: which embeds establish a subtype relation, and under which
// base identity.
func TestSubtypeIndex(t *testing.T) {
	b := newSubtypesBuilder(t)
	idx := b.subtypes()

	t.Run("subtypes of a base are collected across packages", func(t *testing.T) {
		assert.Equal(t,
			[]string{
				subtypesFixtureRoot + "/modelS",
				subtypesFixtureRoot + "/modelX",
				subtypesFixtureRoot + "/sub/modelY",
			},
			declKeys(idx[teslaCarIdentity]),
			"a subtype is indexed wherever it is declared, not only in the base's package")
	})

	t.Run("the index is built regardless of the base carrying a discriminator", func(t *testing.T) {
		// The discriminated gate lives in discriminatedSubtypesOf, not in the index: the index is a pure source fact.
		assert.Equal(t, []string{subtypesFixtureRoot + "/PlainSub"}, declKeys(idx[plainBaseIdentity]))
	})

	t.Run("a pointer embed and an alias embed both resolve to the base", func(t *testing.T) {
		assert.Equal(t,
			[]string{
				subtypesFixtureRoot + "/edges/Bike",  // *Vehicle
				subtypesFixtureRoot + "/edges/Trike", // VehicleAlias
			},
			declKeys(idx[vehicleIdentity]))
	})

	t.Run("an alias embed is also indexed under the alias's own identity", func(t *testing.T) {
		// Which of the two the emitted allOf member $refs depends on RefAliases / TransparentAliases, so the relation is
		// recorded under both.
		assert.Equal(t, []string{subtypesFixtureRoot + "/edges/Trike"}, declKeys(idx[vehicleAliasID]))
	})

	t.Run("an ignored embed establishes no relation", func(t *testing.T) {
		assert.NotContains(t, declKeys(idx[vehicleIdentity]), subtypesFixtureRoot+"/edges/Hidden",
			"swagger:ignore drops the embed here exactly as it does in the schema builder")
	})

	t.Run("a plain (unannotated) embed is not a subtype", func(t *testing.T) {
		assert.NotContains(t, declKeys(idx[vehicleIdentity]), subtypesFixtureRoot+"/edges/Plain",
			"a plain embed inlines properties; no allOf member, no subtype relation")
	})

	t.Run("a model with no allOf embed is absent from the index", func(t *testing.T) {
		for identity, decls := range idx {
			assert.NotContainsf(t, declKeys(decls), subtypesFixtureRoot+"/Unrelated",
				"Unrelated embeds nothing yet appears under %q", identity)
		}
	})

	t.Run("the index is memoised", func(t *testing.T) {
		second := b.subtypes()
		require.NotNil(t, b.subtypeIdx)
		assert.Len(t, second, len(idx))
	})
}

// TestDiscriminatedSubtypesOf locks the discriminated gate and the already-emitted filter of the discovery-side pull.
func TestDiscriminatedSubtypesOf(t *testing.T) {
	b := newSubtypesBuilder(t)
	base := findModelDecl(t, b, "TeslaCar")

	t.Run("a base with no definition yet pulls nothing", func(t *testing.T) {
		assert.Empty(t, b.discriminatedSubtypesOf(base),
			"the gate reads the definition just built; there is none")
	})

	t.Run("a definition without a discriminator pulls nothing", func(t *testing.T) {
		b.definitions[base.DefKey()] = oaispec.Schema{}
		assert.Empty(t, b.discriminatedSubtypesOf(base),
			"allOf users of a plain base are ordinary compositions, not a family")
	})

	t.Run("a discriminated base pulls its whole family", func(t *testing.T) {
		b.definitions[base.DefKey()] = oaispec.Schema{
			SwaggerSchemaProps: oaispec.SwaggerSchemaProps{Discriminator: "model"},
		}
		assert.Equal(t,
			[]string{
				subtypesFixtureRoot + "/modelS",
				subtypesFixtureRoot + "/modelX",
				subtypesFixtureRoot + "/sub/modelY",
			},
			declKeys(b.discriminatedSubtypesOf(base)))
	})

	t.Run("under ScanModels the pull is a no-op", func(t *testing.T) {
		// Every model is built up front there, so pulling would add nothing and would emit Hints in model- index iteration
		// order.
		// The -m case is served by the prune reachability rule.
		withModels := NewBuilder(nil, b.ctx, true)
		withModels.definitions[base.DefKey()] = oaispec.Schema{
			SwaggerSchemaProps: oaispec.SwaggerSchemaProps{Discriminator: "model"},
		}
		assert.Empty(t, withModels.discriminatedSubtypesOf(base))
	})

	t.Run("an already-emitted subtype is not pulled again", func(t *testing.T) {
		b.definitions[subtypesFixtureRoot+"/modelS"] = oaispec.Schema{}
		assert.Equal(t,
			[]string{
				subtypesFixtureRoot + "/modelX",
				subtypesFixtureRoot + "/sub/modelY",
			},
			declKeys(b.discriminatedSubtypesOf(base)))
	})
}

// TestSubtypeIndex_Nested locks the multi-level hierarchy in the index: an INTERFACE that embeds a discriminated
// interface is a subtype too, and is itself a base for the structs below it.
func TestSubtypeIndex_Nested(t *testing.T) {
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages: []string{"./enhancements/discriminated-subtypes-nested/..."},
		WorkDir:  "../../../fixtures",
	})
	require.NoError(t, err)
	idx := NewBuilder(nil, ctx, false).subtypes()

	const nested = "github.com/go-openapi/codescan/fixtures/enhancements/discriminated-subtypes-nested"

	t.Run("an interface embed establishes a subtype relation", func(t *testing.T) {
		// A struct embeds its base as an anonymous field, an interface as an anonymous interface — the index must read both
		// member lists, or a hierarchy stops at its first level.
		assert.Equal(t,
			[]string{nested + "/Circle", nested + "/Polygon"},
			declKeys(idx[nested+".Shape"]),
			"the root's subtypes are the struct leaf and the mid-level interface")
	})

	t.Run("a mid-level type is a base in its own right", func(t *testing.T) {
		assert.Equal(t,
			[]string{nested + "/Square", nested + "/Triangle"},
			declKeys(idx[nested+".Polygon"]))
	})
}

// TestIsDiscriminated locks the gate shared by both hooks: where a definition may declare a discriminator, and what
// must NOT count as one.
func TestIsDiscriminated(t *testing.T) {
	discriminated := func(name string) oaispec.Schema {
		return oaispec.Schema{SwaggerSchemaProps: oaispec.SwaggerSchemaProps{Discriminator: name}}
	}
	refTo := func(target string) oaispec.Schema {
		ref, err := oaispec.NewRef(target)
		require.NoError(t, err)

		return oaispec.Schema{SchemaProps: oaispec.SchemaProps{Ref: ref}}
	}

	t.Run("nil is not discriminated", func(t *testing.T) {
		assert.False(t, isDiscriminated(nil))
	})

	t.Run("a plain base declares it at the top level", func(t *testing.T) {
		sch := discriminated("kind")
		assert.True(t, isDiscriminated(&sch))
	})

	t.Run("a mid-level base declares it inside its own allOf member", func(t *testing.T) {
		// This is the emitted shape of a subtype that is itself a base: its properties (and hence its discriminator) land in
		// the compound member, never at the top.
		sch := oaispec.Schema{SchemaProps: oaispec.SchemaProps{
			AllOf: []oaispec.Schema{refTo("#/definitions/Root"), discriminated("polygonType")},
		}}
		assert.True(t, isDiscriminated(&sch))
	})

	t.Run("a leaf subtype does not inherit its base's discriminator", func(t *testing.T) {
		// The $ref member is not followed, so pointing at a discriminated base does not make the leaf a base — otherwise
		// every subtype would pull its siblings.
		sch := oaispec.Schema{SchemaProps: oaispec.SchemaProps{
			AllOf: []oaispec.Schema{refTo("#/definitions/Root"), {}},
		}}
		assert.False(t, isDiscriminated(&sch))
	})
}

// TestSubtypeKeysOf locks the prune-side bridge: definition key -> Go type identity -> subtype definition keys.
func TestSubtypeKeysOf(t *testing.T) {
	b := newSubtypesBuilder(t)
	base := findModelDecl(t, b, "TeslaCar")

	t.Run("an unknown definition key resolves to no subtypes", func(t *testing.T) {
		assert.Empty(t, b.subtypeKeysOf(base.DefKey()),
			"nothing was built, so no identity was recorded")
	})

	t.Run("a built definition resolves through its recorded identity", func(t *testing.T) {
		b.declIdentity[base.DefKey()] = typeIdentity(base.Obj())
		keys := b.subtypeKeysOf(base.DefKey())
		sort.Strings(keys)
		assert.Equal(t,
			[]string{
				subtypesFixtureRoot + "/modelS",
				subtypesFixtureRoot + "/modelX",
				subtypesFixtureRoot + "/sub/modelY",
			},
			keys)
	})
}
