// SPDX-License-Identifier: Apache-2.0

package maps

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

func scanMaps(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./concepts/maps"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenJSON marshals one definition and compares it to (or, under
// UPDATE_GOLDEN, rewrites) testdata/<feature>.json.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenJSON(t *testing.T, doc *spec.Swagger, feature, defName string) {
	t.Helper()
	schema, ok := doc.Definitions[defName]
	require.Truef(t, ok, "definition %q not found", defName)
	got, err := json.MarshalIndent(schema, "", "  ")
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

// TestMapFragments emits and verifies the golden fragments the tutorial pairs
// with each source region, and asserts the central claims of each form.
func TestMapFragments(t *testing.T) {
	doc := scanMaps(t)

	goldenJSON(t, doc, "naturalmap", "Inventory")
	goldenJSON(t, doc, "keytypes", "Lookups")
	goldenJSON(t, doc, "closed", "ClosedObject")
	goldenJSON(t, doc, "open", "OpenObject")
	goldenJSON(t, doc, "typed", "TypedObject")
	goldenJSON(t, doc, "ref", "RefObject")
	goldenJSON(t, doc, "fieldkeyword", "Holder")
	goldenJSON(t, doc, "patterntyped", "TypedPatterns")
	goldenJSON(t, doc, "addlpropstyped", "Settings")

	// A map renders as an object with a single value schema.
	counts := doc.Definitions["Inventory"].Properties["counts"]
	assert.Equal(t, []string{"object"}, []string(counts.Type))
	require.NotNil(t, counts.AdditionalProperties)

	// #2251: an integer-keyed map is still object+additionalProperties.
	byCode := doc.Definitions["Lookups"].Properties["byCode"]
	assert.Equal(t, []string{"object"}, []string(byCode.Type))
	require.NotNil(t, byCode.AdditionalProperties)

	// false closes the object (Allows=false, no schema).
	closed := doc.Definitions["ClosedObject"]
	require.NotNil(t, closed.AdditionalProperties)
	assert.False(t, closed.AdditionalProperties.Allows)
	assert.Nil(t, closed.AdditionalProperties.Schema)

	// A $ref'd field with additionalProperties rides an allOf sibling.
	closedRef := doc.Definitions["Holder"].Properties["closedRef"]
	require.Len(t, closedRef.AllOf, 2)
	assert.Equal(t, "#/definitions/Thing", closedRef.AllOf[0].Ref.String())

	// The typed patternProperties marker maps a regex to a $ref value.
	tp := doc.Definitions["TypedPatterns"]
	require.Contains(t, tp.PatternProperties, "^item-")
	itemPat := tp.PatternProperties["^item-"]
	assert.Equal(t, "#/definitions/Thing", itemPat.Ref.String())
}
