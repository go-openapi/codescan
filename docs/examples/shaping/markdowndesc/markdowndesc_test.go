// SPDX-License-Identifier: Apache-2.0

package markdowndesc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// scanMarkdownDesc scans the witness tree (no option needed — the literal block
// marker is annotation syntax, opt-in per annotation).
func scanMarkdownDesc(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/markdowndesc/..."},
		ScanModels: true,
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

// TestMarkdownDesc contrasts the two forms end to end: Option B drops the table
// (truncates at the first blank line), while swagger:description | carries the
// whole markdown body — table pipes, blank lines, bullets and indentation —
// verbatim, with the | marker never leaking.
func TestMarkdownDesc(t *testing.T) {
	doc := scanMarkdownDesc(t)

	plain, ok := doc.Definitions["Plain"]
	require.True(t, ok, "Plain definition missing")
	goldenRaw(t, "plain", plain)

	md, ok := doc.Definitions["Markdown"]
	require.True(t, ok, "Markdown definition missing")
	goldenRaw(t, "markdown", md)

	// Plain: Option B folds up to the first blank line — the table is dropped.
	assert.Contains(t, plain.Description, "only this sentence")
	assert.NotContains(t, plain.Description, "| name | purpose |", "Option B drops the post-blank-line table")

	// Markdown: the body is captured verbatim.
	assert.Contains(t, md.Description, "**verbatim**")
	assert.Contains(t, md.Description, "| name | purpose |", "table leading pipes survive")
	assert.Contains(t, md.Description, "\n\n", "significant blank line preserved")
	assert.Contains(t, md.Description, "- point one", "bullets after a blank line are kept")
	assert.NotContains(t, md.Description, "swagger:description", "the marker line never leaks")
	assert.False(t, strings.HasPrefix(md.Description, "|"), "the | marker must not leak into the body")

	// Title still comes from the godoc preamble, not the literal body.
	assert.Equal(t, "Markdown opts into a verbatim body with the literal block marker.", md.Title)

	// Field description keeps the indentation of the ordered list.
	name := md.Properties["name"]
	assert.Contains(t, name.Description, "  1. unique")
	assert.Contains(t, name.Description, "  2. lowercase")
}
