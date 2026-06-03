// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_AliasEmbedInline witnesses the Q-D fix: a struct that
// embeds an alias of a named type must produce the same flat inline
// shape as a struct that embeds the named type directly. The
// documented contract is "embeds always inline properties, never
// $ref unless the embed is swagger:allOf-tagged"
// (internal/builders/schema/embedded.go). The previous code violated
// that contract for aliased embeds — they were silently promoted to
// allOf composition without an explicit annotation.
//
// Discovered during the W3 alias workshop cycle 3 (Q8 confirmation +
// resolution). The cycle-3 fixture types are designed for this
// regression: same Base struct embedded direct, via alias, via
// pointer, and via interface. Post-fix, all four produce structurally
// equivalent inline shapes.
func TestCoverage_AliasEmbedInline(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-calibration-embed/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// EmbedsDirectStruct (Base embed) is the baseline.
	require.Contains(t, doc.Definitions, "EmbedsDirectStruct")
	direct := doc.Definitions["EmbedsDirectStruct"]
	assert.Empty(t, direct.AllOf,
		"direct named embed produces flat inline, never allOf")
	assert.Contains(t, direct.Properties, "id")
	assert.Contains(t, direct.Properties, "name")
	assert.Contains(t, direct.Properties, "extra")

	// EmbedsAlias (BaseAlias = Base embed) must match the direct shape.
	require.Contains(t, doc.Definitions, "EmbedsAlias")
	aliased := doc.Definitions["EmbedsAlias"]
	assert.Empty(t, aliased.AllOf,
		"aliased embed must produce flat inline like direct (Q-D fix)")
	assert.Contains(t, aliased.Properties, "id")
	assert.Contains(t, aliased.Properties, "name")
	assert.Contains(t, aliased.Properties, "extra")

	// EmbedsPointer (*Base embed) — pointer is peeled.
	require.Contains(t, doc.Definitions, "EmbedsPointer")
	ptr := doc.Definitions["EmbedsPointer"]
	assert.Empty(t, ptr.AllOf)
	assert.Contains(t, ptr.Properties, "id")
	assert.Contains(t, ptr.Properties, "name")
	assert.Contains(t, ptr.Properties, "extra")

	// EmbedsInterface (named interface embed) — flat with method
	// promoted as property + the outer's own field.
	require.Contains(t, doc.Definitions, "EmbedsInterface")
	iface := doc.Definitions["EmbedsInterface"]
	assert.Empty(t, iface.AllOf)
	assert.Contains(t, iface.Properties, "describe")
	assert.Contains(t, iface.Properties, "tag")
}

// TestCoverage_AliasEmbedAllOfOptIn witnesses the OTHER half of the
// Q-D contract: when the user EXPLICITLY annotates an aliased embed
// with `swagger:allOf`, the spec produces allOf composition with a
// $ref to the embedded type. This is the shape that was silently
// produced (without annotation) before the Q-D fix; it MUST remain
// reachable as an explicit opt-in.
//
// Combined with TestCoverage_AliasEmbedInline, this asserts the
// bidirectional contract: annotation is the sole gate of allOf
// composition.
func TestCoverage_AliasEmbedAllOfOptIn(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-calibration-embed/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// EmbedsAliasOptIn (BaseAlias embed with swagger:allOf) →
	// allOf: [{$ref: BaseAlias}, {properties: {extra inline}}].
	// Same shape that was silently produced for aliased embeds
	// before the Q-D fix; explicit annotation preserves it.
	require.Contains(t, doc.Definitions, "EmbedsAliasOptIn")
	optInAlias := doc.Definitions["EmbedsAliasOptIn"]
	require.Len(t, optInAlias.AllOf, 2,
		"swagger:allOf on an aliased embed must produce 2-member allOf composition")

	foundRefToAlias := false
	foundInlineExtra := false
	for _, member := range optInAlias.AllOf {
		switch {
		case member.Ref.String() != "":
			assert.Equal(t, "#/definitions/BaseAlias", member.Ref.String(),
				"$ref must point at the aliased type's own definition")
			foundRefToAlias = true
		case len(member.Properties) > 0:
			assert.Contains(t, member.Properties, "extra",
				"outer struct's own fields must appear inline alongside the $ref")
			foundInlineExtra = true
		}
	}
	assert.True(t, foundRefToAlias, "must find $ref member")
	assert.True(t, foundInlineExtra, "must find inline-properties member")

	// EmbedsDirectStructOptIn (Base direct embed with swagger:allOf)
	// → allOf with $ref to Base. Same annotation, same composition
	// shape, regardless of whether the embed was aliased — Q-D
	// makes alias and direct paths agree everywhere.
	require.Contains(t, doc.Definitions, "EmbedsDirectStructOptIn")
	optInDirect := doc.Definitions["EmbedsDirectStructOptIn"]
	require.Len(t, optInDirect.AllOf, 2)

	foundRefToBase := false
	foundInlineExtraDirect := false
	for _, member := range optInDirect.AllOf {
		switch {
		case member.Ref.String() != "":
			assert.Equal(t, "#/definitions/Base", member.Ref.String())
			foundRefToBase = true
		case len(member.Properties) > 0:
			assert.Contains(t, member.Properties, "extra")
			foundInlineExtraDirect = true
		}
	}
	assert.True(t, foundRefToBase)
	assert.True(t, foundInlineExtraDirect)
}
