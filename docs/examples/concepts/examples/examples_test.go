// SPDX-License-Identifier: Apache-2.0

package examples

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

func scanExamples(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./concepts/examples"},
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
	goldenRaw(t, feature, schema)
}

// goldenRaw marshals an arbitrary value (a definition, a response, …) into
// testdata/<feature>.json, honouring UPDATE_GOLDEN.
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

// TestExampleFragments emits and verifies the golden fragments the tutorial
// pairs with each source region. It also confirms example/default values are
// coerced to the field's type rather than left as strings.
func TestExampleFragments(t *testing.T) {
	doc := scanExamples(t)

	goldenJSON(t, doc, "example", "Greeting") // example: values, typed
	goldenJSON(t, doc, "default", "Settings") // default: values, typed
	goldenJSON(t, doc, "reffield", "Price")   // example/default on a $ref'd field (allOf override)

	ntp, ok := doc.Responses["ntpServers"]
	require.True(t, ok, "ntpServers response missing")
	goldenRaw(t, "responseexample", ntp) // example on a top-level array response

	goldenJSON(t, doc, "complexexample", "Profile") // structured (object/array) example values
	goldenJSON(t, doc, "refstructured", "Place")    // JSON-object example coerced on a $ref'd field

	pet, ok := doc.Responses["petResponse"]
	require.True(t, ok, "petResponse response missing")
	goldenRaw(t, "responseexamplesbymime", pet) // response examples keyed by media type

	// The struct-based swagger:response carries a per-media-type examples map.
	require.NotNil(t, pet.Examples, "petResponse must carry an examples map")
	assert.Equal(t, map[string]any{"name": "Fluffy"}, pet.Examples["application/json"])
	assert.Equal(t, "<pet><name>Fluffy</name></pet>", pet.Examples["application/xml"])

	// Type coercion: a numeric default on an int field is a JSON number, a
	// boolean default a JSON bool — not strings.
	port := doc.Definitions["Settings"].Properties["port"]
	assert.EqualValues(t, 8080, port.Default)
	verbose := doc.Definitions["Settings"].Properties["verbose"]
	assert.Equal(t, false, verbose.Default)

	// G3: a JSON-object example on a $ref'd field is coerced into a structured
	// value on the allOf override arm, not carried as a raw string.
	at := doc.Definitions["Place"].Properties["at"]
	require.Len(t, at.AllOf, 2, "expected a $ref arm and an override arm")
	assert.IsType(t, map[string]any{}, at.AllOf[1].Example,
		"object-literal example coerces to a structured value, not a string")

	goldenRaw(t, "full", doc) // whole spec for the tutorial's live "SwaggerUI" tab
}
