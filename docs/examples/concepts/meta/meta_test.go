// SPDX-License-Identifier: Apache-2.0

package meta

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

func scanMeta(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:  examplesRoot(t),
		Packages: []string{"./concepts/meta"},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestMetaFragment emits and verifies testdata/meta.json — the whole document a
// swagger:meta-only package produces, which the "Document metadata" tutorial
// renders next to the package doc comment.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestMetaFragment(t *testing.T) {
	doc := scanMeta(t)
	require.NotNil(t, doc.Info)
	assert.Equal(t, "Pet Store.", doc.Info.Title) // title strips the "Package meta" prefix

	got, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", "meta.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}
