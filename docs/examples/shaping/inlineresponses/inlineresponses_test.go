// SPDX-License-Identifier: Apache-2.0

package inlineresponses

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

func scanInlineResponses(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/inlineresponses"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestInlineResponses emits and verifies testdata/pathitem.json — the GET /pets
// path item the "Inline response bodies" guide renders — and asserts the body:
// sub-language produces a primitive, an array-of-$ref, and a single $ref body
// without any swagger:response struct.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestInlineResponses(t *testing.T) {
	doc := scanInlineResponses(t)

	require.NotNil(t, doc.Paths)
	pets, ok := doc.Paths.Paths["/pets"]
	require.True(t, ok, "GET /pets path missing")
	require.NotNil(t, pets.Get)

	resps := pets.Get.Responses.StatusCodeResponses
	require.Contains(t, resps, 200)
	require.Contains(t, resps, 400)
	assert.Equal(t, "string", resps[400].Schema.Type[0], "400 body: a primitive string schema")

	got, err := json.MarshalIndent(pets, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", "pathitem.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}
