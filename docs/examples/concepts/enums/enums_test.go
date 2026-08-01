// SPDX-License-Identifier: Apache-2.0

package enums

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

func scanEnums(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./concepts/enums"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenJSON marshals v and compares it to (or, under UPDATE_GOLDEN, rewrites)
// testdata/<feature>.json — the fragment the "Enumerations" tutorial renders
// next to the annotated source.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenJSON(t *testing.T, feature string, v any) {
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

func definitionOf(t *testing.T, doc *spec.Swagger, name string) spec.Schema {
	t.Helper()
	schema, ok := doc.Definitions[name]
	require.Truef(t, ok, "definition %q not found", name)
	return schema
}

// TestEnumFragments emits and verifies the golden fragments the tutorial pairs
// with each source region: one per shaping rule, plus the simple-schema
// parameter surface where the enum ships inline.
func TestEnumFragments(t *testing.T) {
	doc := scanEnums(t)

	require.NotNil(t, doc.Paths)
	search, ok := doc.Paths.Paths["/cameras/search"]
	require.True(t, ok, "GET /cameras/search missing")
	require.NotNil(t, search.Get)

	goldenJSON(t, "iota", definitionOf(t, doc, "Schedule"))         // values the syntax does not carry
	goldenJSON(t, "expressions", definitionOf(t, doc, "Threshold")) // computed members
	goldenJSON(t, "signed", definitionOf(t, doc, "Camera"))         // negative member, int8 width
	goldenJSON(t, "width", definitionOf(t, doc, "Lens"))            // float32 typed from the declaration
	goldenJSON(t, "strfmt", definitionOf(t, doc, "Label"))          // format carried over from strfmt.UUID
	goldenJSON(t, "runes", definitionOf(t, doc, "Glyph"))           // rune members are code points
	goldenJSON(t, "params", search.Get.Parameters)                  // inline on a non-body parameter

	goldenJSON(t, "full", doc) // whole spec for the tutorial's live "SwaggerUI" tab
}

// TestDeclarationOrderDoesNotDecideTheType pins the rule the tutorial states in
// prose: Zoom's first member is written as an integer literal, and the schema is
// a number enum all the same, because the type comes from the declaration.
func TestDeclarationOrderDoesNotDecideTheType(t *testing.T) {
	doc := scanEnums(t)

	zoom := definitionOf(t, doc, "Lens").Properties["zoom"]
	assert.Equal(t, []string{"number"}, []string(zoom.Type))
	assert.Equal(t, "float", zoom.Format)
}
