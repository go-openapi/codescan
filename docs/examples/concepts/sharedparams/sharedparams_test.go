// SPDX-License-Identifier: Apache-2.0

package sharedparams

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

func scanSharedParams(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:  examplesRoot(t),
		Packages: []string{"./concepts/sharedparams"},
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

// paramRefs collects the #/parameters/{name} $ref targets on a parameter slice.
func paramRefs(params []spec.Parameter) []string {
	var refs []string
	for i := range params {
		if r := params[i].Ref.String(); r != "" {
			refs = append(refs, r)
		}
	}
	return refs
}

// TestSharedParameters emits and verifies the golden fragments the tutorial
// pairs with each source region, and pins the behaviours the prose claims.
func TestSharedParameters(t *testing.T) {
	doc := scanSharedParams(t)
	require.NotNil(t, doc.Paths)

	// --- The shared namespace (#/parameters, #/responses) -----------------
	goldenRaw(t, "parameters", doc.Parameters)
	goldenRaw(t, "responses", doc.Responses)

	// `swagger:parameters *` registers a top-level header parameter; the
	// `* createPet` form registers AND is wired below.
	reqID, ok := doc.Parameters["X-Request-ID"]
	require.True(t, ok, "expected #/parameters/X-Request-ID")
	assert.Equal(t, "header", reqID.In)
	apiKey, ok := doc.Parameters["X-API-Key"]
	require.True(t, ok, "expected #/parameters/X-API-Key")
	assert.True(t, apiKey.Required)
	// Inline-only params (the createPet body) do not leak into #/parameters.
	_, hasBody := doc.Parameters["body"]
	assert.False(t, hasBody, "inline operation params must not leak into #/parameters")

	// `swagger:response *` registers a shared response.
	_, ok = doc.Responses["ErrorResponse"]
	require.True(t, ok, "expected #/responses/ErrorResponse from swagger:response *")

	// --- Operations: both reference channels ------------------------------
	pets, ok := doc.Paths.Paths["/pets"]
	require.True(t, ok, "expected a /pets path")
	goldenRaw(t, "listpets", pets.Get)
	goldenRaw(t, "createpet", pets.Post)

	// listPets: standalone `swagger:parameters listPets X-Request-ID` marker.
	require.NotNil(t, pets.Get)
	assert.Contains(t, paramRefs(pets.Get.Parameters), "#/parameters/X-Request-ID",
		"listPets should $ref the shared X-Request-ID (standalone marker)")
	// createPet: `* createPet` convenience form.
	require.NotNil(t, pets.Post)
	assert.Contains(t, paramRefs(pets.Post.Parameters), "#/parameters/X-API-Key",
		"createPet should $ref the shared X-API-Key (via `* createPet`)")

	// Both routes resolve `default: ErrorResponse` to a $ref.
	for _, op := range []*spec.Operation{pets.Get, pets.Post} {
		require.NotNil(t, op.Responses)
		require.NotNil(t, op.Responses.Default)
		assert.Equal(t, "#/responses/ErrorResponse", op.Responses.Default.Ref.String())
	}

	// --- Path-item parameters (exact path, no hierarchy) ------------------
	petByID, ok := doc.Paths.Paths["/pets/{id}"]
	require.True(t, ok, "expected a /pets/{id} path")
	goldenRaw(t, "pathitem", petByID)

	var hasTenant bool
	for i := range petByID.Parameters {
		if petByID.Parameters[i].Name == "X-Tenant" && petByID.Parameters[i].In == "header" {
			hasTenant = true
		}
	}
	assert.True(t, hasTenant, "expected the inline X-Tenant header on the /pets/{id} path-item")

	// Exact path: the /pets/{id} path-item header must NOT leak onto /pets.
	for i := range pets.Parameters {
		assert.NotEqual(t, "X-Tenant", pets.Parameters[i].Name,
			"/pets must not inherit /pets/{id} path-item parameters (no hierarchy)")
	}
}
