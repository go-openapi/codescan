// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func runDefaultAllOfEmbeds(t *testing.T, on bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		Packages:              []string{"./enhancements/default-allof-embeds/..."},
		WorkDir:               scantest.FixturesDir(),
		ScanModels:            true,
		DefaultAllOfForEmbeds: on,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestCoverage_DefaultAllOfEmbeds_Off is the control: with the flag off a plain struct embed
// inlines the embedded type's properties (the historic behaviour).
func TestCoverage_DefaultAllOfEmbeds_Off(t *testing.T) {
	doc := runDefaultAllOfEmbeds(t, false)

	pe := doc.Definitions["PlainEmbed"]
	// Inlined flat: embedded fields land directly as properties, no allOf.
	assert.Empty(t, pe.AllOf, "plain embed must not compose when the flag is off")
	assert.Contains(t, pe.Properties, "id", "model embed inlined")
	assert.Contains(t, pe.Properties, "name", "model embed inlined")
	assert.Contains(t, pe.Properties, "note", "non-model embed inlined")
	assert.Contains(t, pe.Properties, "color", "own field present")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_default_allof_embeds_off.json")
}

// TestCoverage_DefaultAllOfEmbeds_On exercises the feature: a plain model embed becomes an allOf
// $ref member, a plain non-model embed an inline allOf member, and the embedding struct's own
// fields a sibling allOf member.
//
// A json-named embed and an explicit swagger:allOf embed are unaffected.
func TestCoverage_DefaultAllOfEmbeds_On(t *testing.T) {
	doc := runDefaultAllOfEmbeds(t, true)

	// PlainEmbed composes; no field is inlined at the top level.
	pe := doc.Definitions["PlainEmbed"]
	require.NotEmpty(t, pe.AllOf, "plain embed must compose into allOf when the flag is on")
	assert.NotContains(t, pe.Properties, "id", "model embed must not inline when composing")
	assert.NotContains(t, pe.Properties, "color", "own fields move into a sibling allOf member")
	assert.True(t, allOfHasRef(pe, "#/definitions/Base"), "model embed becomes a $ref allOf member")
	assert.True(t, allOfHasProp(pe, "note"), "non-model embed becomes an inline allOf member")
	assert.True(t, allOfHasProp(pe, "color"), "own field lands in a sibling allOf member")

	// PointerEmbed: *Base peeled, same $ref composition.
	ptr := doc.Definitions["PointerEmbed"]
	require.NotEmpty(t, ptr.AllOf)
	assert.True(t, allOfHasRef(ptr, "#/definitions/Base"), "pointer embed peels to a $ref member")
	assert.True(t, allOfHasProp(ptr, "tag"), "own field in sibling member")

	// NamedEmbed: a json-named embed stays a single nested property, on or off.
	named := doc.Definitions["NamedEmbed"]
	assert.Empty(t, named.AllOf, "a json-named embed must not compose")
	assert.Contains(t, named.Properties, "base", "named embed nests under its json name")
	assert.Contains(t, named.Properties, "extra")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_default_allof_embeds_on.json")
}

// allOfHasRef reports whether any allOf member of sch is a $ref to ref.
func allOfHasRef(sch spec.Schema, ref string) bool {
	for _, m := range sch.AllOf {
		if m.Ref.String() == ref {
			return true
		}
	}
	return false
}

// allOfHasProp reports whether any allOf member of sch carries property name.
func allOfHasProp(sch spec.Schema, name string) bool {
	for _, m := range sch.AllOf {
		if _, ok := m.Properties[name]; ok {
			return true
		}
	}
	return false
}
