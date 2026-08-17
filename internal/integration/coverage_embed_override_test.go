// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const embedOverridePkg = "./enhancements/default-allof-embeds-override/..."

// runEmbedOverride scans the override fixture in one of the two embed renderings.
func runEmbedOverride(t *testing.T, defaultAllOf bool) *oaispec.Swagger {
	t.Helper()
	doc, err := runScan(&codescan.Options{
		Packages:              []string{embedOverridePkg},
		WorkDir:               scantest.FixturesDir(),
		DefaultAllOfForEmbeds: defaultAllOf,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	return doc
}

// TestEmbedOverride_Inlined locks the faithful case: inlining RESOLVES an override, applying Go's
// depth rule, so the schema describes the type the compiler sees.
//
// Go marshals exactly one field per shadowed name — the outer one — and that is what is emitted.
func TestEmbedOverride_Inlined(t *testing.T) {
	doc := runEmbedOverride(t, false)

	t.Run("a decoration on the outer declaration wins", func(t *testing.T) {
		id := doc.Definitions["Decorated"].Properties["ID"]
		assert.Equal(t, oaispec.StringOrArray{"integer"}, id.Type)
		assert.True(t, id.ReadOnly)
	})

	t.Run("a retyped override wins outright", func(t *testing.T) {
		assert.Equal(t, oaispec.StringOrArray{"string"}, doc.Definitions["Retyped"].Properties["ID"].Type,
			"the outer string declaration is the one Go marshals")
	})

	t.Run("a differently-named override adds a property rather than replacing one", func(t *testing.T) {
		// Go emits BOTH here: the JSON names differ, so neither shadows the other.
		renamed := doc.Definitions["Renamed"]
		assert.Contains(t, renamed.Properties, "Name")
		assert.Contains(t, renamed.Properties, "fullName")
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_embed_override.json")
}

// TestEmbedOverride_DefaultAllOf documents the limit of composition, and why `swagger:omit` exists.
//
// allOf ACCUMULATES: members conjoin, and a conjunction can only narrow, never replace. An override
// that replaces is therefore not expressible — the base member keeps the shadowed declaration:
//
//   - Decorated: `ID` appears in both members (valid, but redundant — a
//     generator walking members sees it twice);
//   - Retyped: `ID` must be integer AND string, which nothing can satisfy.
//
// codescan does not guess the author's intent here (there are many ways to write a Go type whose
// composition cannot be expressed); `swagger:omit` on the embed is how the author resolves it —
// see TestSwaggerOmit_DefaultAllOf for the same shapes, resolved.
func TestEmbedOverride_DefaultAllOf(t *testing.T) {
	doc := runEmbedOverride(t, true)

	t.Run("a decorated override is carried by both members", func(t *testing.T) {
		decorated := doc.Definitions["Decorated"]
		require.Len(t, decorated.AllOf, 2)
		assert.Contains(t, decorated.AllOf[0].Properties, "ID", "the base member keeps the shadowed field")
		assert.True(t, decorated.AllOf[1].Properties["ID"].ReadOnly)
	})

	t.Run("a retyped override yields a schema nothing can satisfy", func(t *testing.T) {
		retyped := doc.Definitions["Retyped"]
		require.Len(t, retyped.AllOf, 2)
		assert.Equal(t, oaispec.StringOrArray{"integer"}, retyped.AllOf[0].Properties["ID"].Type)
		assert.Equal(t, oaispec.StringOrArray{"string"}, retyped.AllOf[1].Properties["ID"].Type,
			"integer AND string — the documented limit that swagger:omit resolves")
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_embed_override_allof.json")
}
