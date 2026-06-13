// SPDX-License-Identifier: Apache-2.0

package aliases

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

func scanAliases(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/aliases"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestAliasDissolves emits testdata/invoice.json and asserts the default
// behavior: an unannotated alias dissolves to its target, so the Price field
// resolves to a $ref to Money and Price gets no definition of its own.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestAliasDissolves(t *testing.T) {
	doc := scanAliases(t)

	_, hasPrice := doc.Definitions["Price"]
	assert.False(t, hasPrice, "unannotated alias Price must not produce its own definition")

	invoice, ok := doc.Definitions["Invoice"]
	require.True(t, ok, "Invoice definition missing")

	got, err := json.MarshalIndent(invoice, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", "invoice.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}
