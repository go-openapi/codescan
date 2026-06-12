// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// examplesRoot returns the absolute path to the examples module root, so the
// scan resolves "./petstore" regardless of the test's working directory.
func examplesRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// thisFile is <root>/basic/scan_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
}

// TestScanPetstore verifies the generated spec against the golden file shown on
// the documentation site (testdata/swagger.json), so the published output can
// never silently drift from what the scanner actually produces.
//
// Regenerate the golden with: UPDATE_GOLDEN=1 go test ./...
func TestScanPetstore(t *testing.T) {
	doc, err := ScanPetstore(examplesRoot(t))
	require.NoError(t, err)
	require.NotNil(t, doc)

	got, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", "swagger.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}

	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}
