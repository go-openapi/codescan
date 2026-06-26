// SPDX-License-Identifier: Apache-2.0

package validations

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

func scanValidations(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./concepts/validations"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenJSON marshals v and compares it to (or, under UPDATE_GOLDEN, rewrites)
// testdata/<feature>.json — the fragment the "Validations" tutorial renders next
// to the annotated source.
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

// TestValidationFragments emits and verifies the golden fragments the tutorial
// pairs with each source region: the full surface on a model field, the reduced
// surface on simple-schema query parameters, and a validated response header.
func TestValidationFragments(t *testing.T) {
	doc := scanValidations(t)

	product, ok := doc.Definitions["Product"]
	require.True(t, ok, "Product definition missing")

	require.NotNil(t, doc.Paths)
	search, ok := doc.Paths.Paths["/products/search"]
	require.True(t, ok, "GET /products/search missing")
	require.NotNil(t, search.Get)

	rate, ok := doc.Responses["rateLimited"]
	require.True(t, ok, "rateLimited response missing")

	attrs, ok := doc.Definitions["Attributes"]
	require.True(t, ok, "Attributes definition missing")

	goldenJSON(t, "field", product)               // full JSON-schema surface
	goldenJSON(t, "param", search.Get.Parameters) // reduced simple-schema surface
	goldenJSON(t, "header", rate)                 // validated response header
	goldenJSON(t, "object", attrs)                // object-validation keywords

	goldenJSON(t, "full", doc) // whole spec for the tutorial's live "SwaggerUI" tab
}
