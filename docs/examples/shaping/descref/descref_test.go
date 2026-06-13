// SPDX-License-Identifier: Apache-2.0

package descref

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/require"
)

func examplesRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func scan(t *testing.T, descWithRef bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:     examplesRoot(t),
		Packages:    []string{"./shaping/descref"},
		ScanModels:  true,
		DescWithRef: descWithRef,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenDef marshals the Person definition and compares it to (or, under
// UPDATE_GOLDEN, rewrites) testdata/<feature>.json.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenDef(t *testing.T, doc *spec.Swagger, feature string) {
	t.Helper()
	schema, ok := doc.Definitions["Person"]
	require.True(t, ok, "Person definition missing")
	got, err := json.MarshalIndent(schema, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", feature+".json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(got))
}

// TestDescWithRef emits the before/after fragments for the description-only
// $ref field.
func TestDescWithRef(t *testing.T) {
	goldenDef(t, scan(t, false), "off") // default: bare $ref, description dropped
	goldenDef(t, scan(t, true), "on")   // DescWithRef: allOf wrapper carries the description
}
