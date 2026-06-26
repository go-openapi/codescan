// SPDX-License-Identifier: Apache-2.0

package namingfromtags

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

func scan(t *testing.T, nameTags []string) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:      examplesRoot(t),
		Packages:     []string{"./shaping/naming-from-tags"},
		ScanModels:   true,
		NameFromTags: nameTags,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenDef marshals the Filter definition and compares it to (or, under
// UPDATE_GOLDEN, rewrites) testdata/<feature>.json.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenDef(t *testing.T, doc *spec.Swagger, feature string) {
	t.Helper()
	schema, ok := doc.Definitions["Filter"]
	require.True(t, ok, "Filter definition missing")
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

// TestNamingFromTags emits the before/after fragments and asserts the toggle:
// property names come from json by default, and from form when NameFromTags
// lists it first.
func TestNamingFromTags(t *testing.T) {
	def := scan(t, nil)                       // default → ["json"]
	form := scan(t, []string{"form", "json"}) // prefer form, fall back to json

	goldenDef(t, def, "default")
	goldenDef(t, form, "form")

	_, hasJSON := def.Definitions["Filter"].Properties["sortKey"]
	assert.True(t, hasJSON, "default: name comes from the json tag")

	_, hasForm := form.Definitions["Filter"].Properties["sort_key"]
	assert.True(t, hasForm, "form-first: name comes from the form tag")
}
