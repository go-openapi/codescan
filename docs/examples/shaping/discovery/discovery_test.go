// SPDX-License-Identifier: Apache-2.0

package discovery

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

func scanDiscovery(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/discovery"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestDiscovery emits and verifies testdata/definitions.json — the definitions
// map the "When the scanner emits a type" guide renders — and asserts the
// reachability rule: referenced types and swagger:model roots appear; an
// unreferenced, unannotated type does not.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestDiscovery(t *testing.T) {
	doc := scanDiscovery(t)

	_, hasOrphan := doc.Definitions["Orphan"]
	assert.False(t, hasOrphan, "unreferenced, unannotated Orphan must not be emitted")
	_, hasOrder := doc.Definitions["Order"]
	assert.True(t, hasOrder, "referenced Order must be emitted even without swagger:model")
	_, hasStandalone := doc.Definitions["Standalone"]
	assert.True(t, hasStandalone, "unreferenced swagger:model Standalone must be emitted under ScanModels")

	got, err := json.MarshalIndent(doc.Definitions, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", "definitions.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}
