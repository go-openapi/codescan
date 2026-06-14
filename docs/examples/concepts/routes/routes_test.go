// SPDX-License-Identifier: Apache-2.0

package routes

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

func scanRoutes(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./concepts/routes"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenJSON marshals v and compares it to (or, under UPDATE_GOLDEN, rewrites)
// testdata/<feature>.json — the fragment the "Routes & operations" tutorial
// renders next to the annotated source.
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

// TestRouteFragments emits and verifies the per-annotation golden fragments the
// tutorial pairs with each source region.
func TestRouteFragments(t *testing.T) {
	doc := scanRoutes(t)
	require.NotNil(t, doc.Paths)
	paths := doc.Paths.Paths

	listPets, ok := paths["/pets"]
	require.True(t, ok, "GET /pets path missing")
	require.NotNil(t, listPets.Get)

	getPet, ok := paths["/pets/{id}"]
	require.True(t, ok, "GET /pets/{id} path missing")

	upload, ok := paths["/pets/{id}/photo"]
	require.True(t, ok, "POST /pets/{id}/photo path missing")
	require.NotNil(t, upload.Post)

	resp, ok := doc.Responses["petsResponse"]
	require.True(t, ok, "petsResponse missing from top-level responses")

	search, ok := paths["/pets/search"]
	require.True(t, ok, "GET /pets/search path missing")
	require.NotNil(t, search.Get)

	catalog, ok := doc.Definitions["CatalogEntry"]
	require.True(t, ok, "CatalogEntry definition missing")

	goldenJSON(t, "route", listPets)                     // swagger:route — a path item
	goldenJSON(t, "operation", getPet)                   // swagger:operation — YAML body
	goldenJSON(t, "parameters", listPets.Get.Parameters) // swagger:parameters — params on the op
	goldenJSON(t, "response", resp)                      // swagger:response — a named response
	goldenJSON(t, "file", upload.Post.Parameters)        // swagger:file — a formData file param
	goldenJSON(t, "externaldocs", search.Get.ExternalDocs) // externalDocs on an operation
	goldenJSON(t, "externaldocs_schema", catalog)          // externalDocs on a full schema
}
