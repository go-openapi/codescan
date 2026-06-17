// SPDX-License-Identifier: Apache-2.0

package formats

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

func scanFormats(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/formats"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestFormats emits and verifies testdata/measurement.json — the definition the
// "Forcing a conformant format" guide renders — and asserts the default uint64
// vendor format vs the swagger:strfmt-forced string/int64 encoding.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestFormats(t *testing.T) {
	doc := scanFormats(t)

	m, ok := doc.Definitions["Measurement"]
	require.True(t, ok, "Measurement definition missing")

	raw := m.Properties["raw"]
	assert.Equal(t, "integer", raw.Type[0])
	assert.Equal(t, "uint64", raw.Format, "raw keeps the Go-derived vendor format")

	bounded := m.Properties["bounded"]
	assert.Equal(t, "string", bounded.Type[0])
	assert.Equal(t, "int64", bounded.Format, "swagger:strfmt forces a conformant string/int64")

	got, err := json.MarshalIndent(m, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", "measurement.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}
