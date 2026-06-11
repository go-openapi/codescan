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

// TestCoverage_AliasEmbedInline pins the embed-handling contract:
// a struct that embeds an alias of a named type produces the same
// flat inline shape as a struct that embeds the named type
// directly. Embeds always inline properties, never `$ref` unless
// the embed carries `swagger:allOf`
// (internal/builders/schema/embedded.go). The fixture exercises
// the same Base struct embedded direct, via alias, via pointer,
// and via interface — all four produce structurally equivalent
// inline shapes.
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
		"aliased embed must produce flat inline like the direct embed")
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

// TestCoverage_AliasEmbedAllOfOptIn pins the other half of the
// embed contract: when the user EXPLICITLY annotates an aliased
// embed with `swagger:allOf`, the spec produces allOf composition
// with a `$ref` to the embedded type. Together with
// TestCoverage_AliasEmbedInline this asserts the bidirectional
// contract: annotation is the sole gate of allOf composition.
func TestCoverage_AliasEmbedAllOfOptIn(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-calibration-embed/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// EmbedsAliasOptIn (BaseAlias embed with swagger:allOf) →
	// allOf: [{$ref: Base}, {properties: {extra inline}}].
	//
	// The `swagger:allOf` annotation governs COMPOSITION SHAPE
	// (allOf vs flat inline). It does not preserve alias identity:
	// the $ref target is the unaliased Base because the alias
	// BaseAlias itself is unannotated (no `swagger:model`). Only
	// `swagger:model` makes an alias a first-class spec entity;
	// without it, the aliasing is a Go implementation detail and
	// the spec carries the underlying name even inside an allOf
	// composition.
	//
	// To preserve the alias name in allOf composition, the alias
	// would need its own `swagger:model` annotation (see
	// EmbedsAliasModeledOptIn below for the witness).
	require.Contains(t, doc.Definitions, "EmbedsAliasOptIn")
	optInAlias := doc.Definitions["EmbedsAliasOptIn"]
	require.Len(t, optInAlias.AllOf, 2,
		"swagger:allOf on an aliased embed must produce 2-member allOf composition")

	foundRefToBase := false
	foundInlineExtra := false
	for _, member := range optInAlias.AllOf {
		switch {
		case member.Ref.String() != "":
			assert.Equal(t, "#/definitions/Base", member.Ref.String(),
				"unannotated alias dissolves to its unaliased target even inside allOf composition")
			foundRefToBase = true
		case len(member.Properties) > 0:
			assert.Contains(t, member.Properties, "extra",
				"outer struct's own fields must appear inline alongside the $ref")
			foundInlineExtra = true
		}
	}
	assert.True(t, foundRefToBase, "must find $ref member pointing at Base")
	assert.True(t, foundInlineExtra, "must find inline-properties member")

	// EmbedsDirectStructOptIn (Base direct embed with swagger:allOf)
	// → allOf with $ref to Base. Same annotation, same composition
	// shape, regardless of whether the embed was aliased — alias
	// and direct paths agree everywhere.
	require.Contains(t, doc.Definitions, "EmbedsDirectStructOptIn")
	optInDirect := doc.Definitions["EmbedsDirectStructOptIn"]
	require.Len(t, optInDirect.AllOf, 2)

	foundRefToBaseDirect := false
	foundInlineExtraDirect := false
	for _, member := range optInDirect.AllOf {
		switch {
		case member.Ref.String() != "":
			assert.Equal(t, "#/definitions/Base", member.Ref.String())
			foundRefToBaseDirect = true
		case len(member.Properties) > 0:
			assert.Contains(t, member.Properties, "extra")
			foundInlineExtraDirect = true
		}
	}
	assert.True(t, foundRefToBaseDirect)
	assert.True(t, foundInlineExtraDirect)

	// EmbedsAliasModeledOptIn (BaseAliasModeled embed with swagger:allOf)
	// → allOf: [{$ref: BaseAliasModeled}, {properties: {extra inline}}].
	//
	// Bidirectional witness against EmbedsAliasOptIn above: same
	// allOf composition shape, same outer struct, but the embedded
	// alias is ANNOTATED. The annotation surfaces the alias name
	// in the $ref target — `swagger:model` is the SOLE knob that
	// preserves alias identity inside an allOf member.
	require.Contains(t, doc.Definitions, "EmbedsAliasModeledOptIn")
	optInAliasModeled := doc.Definitions["EmbedsAliasModeledOptIn"]
	require.Len(t, optInAliasModeled.AllOf, 2,
		"swagger:allOf on an annotated aliased embed must produce 2-member allOf composition")

	foundRefToAliasModeled := false
	foundInlineExtraModeled := false
	for _, member := range optInAliasModeled.AllOf {
		switch {
		case member.Ref.String() != "":
			assert.Equal(t, "#/definitions/BaseAliasModeled", member.Ref.String(),
				"annotated alias preserves identity inside allOf composition")
			foundRefToAliasModeled = true
		case len(member.Properties) > 0:
			assert.Contains(t, member.Properties, "extra")
			foundInlineExtraModeled = true
		}
	}
	assert.True(t, foundRefToAliasModeled, "must find $ref to BaseAliasModeled")
	assert.True(t, foundInlineExtraModeled, "must find inline-properties member")
}
