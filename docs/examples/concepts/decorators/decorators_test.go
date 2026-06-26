// SPDX-License-Identifier: Apache-2.0

package decorators

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

func scanDecorators(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./concepts/decorators"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenJSON marshals v and compares it to (or, under UPDATE_GOLDEN, rewrites)
// testdata/<feature>.json — the fragment the "Other type decorators" tutorial
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

// TestDecoratorFragments emits and verifies the golden fragments the tutorial
// pairs with each source region: readOnly on a model field, and deprecated on
// an operation.
func TestDecoratorFragments(t *testing.T) {
	doc := scanDecorators(t)

	token, ok := doc.Definitions["Token"]
	require.True(t, ok, "Token definition missing")
	require.NotNil(t, doc.Paths)
	ping, ok := doc.Paths.Paths["/legacy/ping"]
	require.True(t, ok, "GET /legacy/ping missing")
	require.NotNil(t, ping.Get)

	gadget, ok := doc.Definitions["Gadget"]
	require.True(t, ok, "Gadget definition missing")

	goldenJSON(t, "readonly", token)          // read only: true → readOnly on the property
	goldenJSON(t, "deprecated", ping)         // deprecated: true on the operation
	goldenJSON(t, "deprecated_model", gadget) // model/field deprecation → x-deprecated

	goldenJSON(t, "full", doc) // whole spec for the tutorial's live "SwaggerUI" tab
}
