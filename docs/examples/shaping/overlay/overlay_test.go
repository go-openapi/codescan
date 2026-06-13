// SPDX-License-Identifier: Apache-2.0

package overlay

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

const baseSpec = `{
  "swagger": "2.0",
  "info": { "title": "Inventory API", "version": "1.0.0" },
  "host": "api.example.com",
  "basePath": "/v1",
  "definitions": {
    "Health": {
      "type": "object",
      "properties": { "ok": { "type": "boolean" } }
    }
  }
}`

func examplesRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func writeGolden(t *testing.T, feature string, v any) {
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

// TestOverlay scans the package on top of a hand-authored base spec and writes
// the before (base) / after (merged) golden fragments. The scan preserves the
// input's metadata and hand-authored definitions and adds the discovered ones.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestOverlay(t *testing.T) {
	var base spec.Swagger
	require.NoError(t, json.Unmarshal([]byte(baseSpec), &base))
	writeGolden(t, "base", &base)

	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/overlay"},
		ScanModels: true,
		InputSpec:  &base,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	writeGolden(t, "merged", doc)

	require.NotNil(t, doc.Info)
	assert.Equal(t, "Inventory API", doc.Info.Title, "input metadata must be preserved")
	_, hasHealth := doc.Definitions["Health"]
	assert.True(t, hasHealth, "hand-authored Health definition must be preserved")
	_, hasWidget := doc.Definitions["Widget"]
	assert.True(t, hasWidget, "scanned Widget definition must be added")
}
