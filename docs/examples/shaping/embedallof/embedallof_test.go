// SPDX-License-Identifier: Apache-2.0

package embedallof

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func examplesRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

// scanEmbedAllOf scans the witness tree with DefaultAllOfForEmbeds off or on.
func scanEmbedAllOf(t *testing.T, on bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:               examplesRoot(t),
		Packages:              []string{"./shaping/embedallof/..."},
		ScanModels:            true,
		DefaultAllOfForEmbeds: on,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenRaw marshals an arbitrary value into testdata/<feature>.json,
// honouring UPDATE_GOLDEN.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenRaw(t *testing.T, feature string, v any) {
	t.Helper()
	got, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", feature+".json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
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

// TestEmbedAllOf_Off is the control: with the flag off a plain embed inlines
// the embedded type's properties (byte-identical to the historic behaviour).
func TestEmbedAllOf_Off(t *testing.T) {
	doc := scanEmbedAllOf(t, false)
	pe, ok := doc.Definitions["PlainEmbed"]
	require.True(t, ok, "PlainEmbed definition missing")
	goldenRaw(t, "plainembed_off", pe)

	// Inlined flat: every embedded + own field lands as a top-level property.
	assert.Empty(t, pe.AllOf, "plain embed must not compose when off")
	for _, p := range []string{"id", "name", "note", "color"} {
		_, has := pe.Properties[p]
		assert.True(t, has, "field %q inlined when off", p)
	}
}

// TestEmbedAllOf_On exercises the feature: a plain model embed becomes an allOf
// $ref member, a plain non-model embed an inline allOf member, and the embedding
// struct's own fields a sibling allOf member. A json-named embed and an explicit
// swagger:allOf embed are unaffected.
func TestEmbedAllOf_On(t *testing.T) {
	doc := scanEmbedAllOf(t, true)
	pe, ok := doc.Definitions["PlainEmbed"]
	require.True(t, ok, "PlainEmbed definition missing")
	goldenRaw(t, "plainembed_on", pe)

	// PlainEmbed composes; no embedded/own field is inlined at the top level.
	require.NotEmpty(t, pe.AllOf, "plain embed must compose into allOf when on")
	_, hasID := pe.Properties["id"]
	assert.False(t, hasID, "model embed must not inline when composing")
	assert.True(t, allOfHasRef(pe, "#/definitions/Base"), "model embed → $ref allOf member")
	assert.True(t, allOfHasProp(pe, "note"), "non-model embed → inline allOf member")
	assert.True(t, allOfHasProp(pe, "color"), "own field → sibling allOf member")

	// PointerEmbed: *Base is peeled to the same $ref composition.
	ptr := doc.Definitions["PointerEmbed"]
	require.NotEmpty(t, ptr.AllOf)
	assert.True(t, allOfHasRef(ptr, "#/definitions/Base"), "pointer embed peels to a $ref member")
	assert.True(t, allOfHasProp(ptr, "tag"), "own field in sibling member")

	// NamedEmbed: a json-named embed stays a single nested property, unaffected.
	named := doc.Definitions["NamedEmbed"]
	assert.Empty(t, named.AllOf, "a json-named embed must not compose")
	_, hasBase := named.Properties["base"]
	assert.True(t, hasBase, "named embed nests under its json name")

	// TaggedEmbed: an explicit swagger:allOf embed is already composition.
	tagged := doc.Definitions["TaggedEmbed"]
	assert.True(t, allOfHasRef(tagged, "#/definitions/Base"), "swagger:allOf embed already composes")
}
