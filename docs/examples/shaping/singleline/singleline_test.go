// SPDX-License-Identifier: Apache-2.0

package singleline

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

func scan(t *testing.T, singleLineAsDesc bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:                        examplesRoot(t),
		Packages:                       []string{"./shaping/singleline"},
		ScanModels:                     true,
		SingleLineCommentAsDescription: singleLineAsDesc,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// fragment captures the title/summary vs description split on both a model and
// an operation, so one golden shows the knob's uniform effect.
func fragment(t *testing.T, doc *spec.Swagger) map[string]any {
	t.Helper()
	gadget, ok := doc.Definitions["Gadget"]
	require.True(t, ok, "Gadget definition missing")
	require.NotNil(t, doc.Paths)
	path, ok := doc.Paths.Paths["/gadgets"]
	require.True(t, ok, "GET /gadgets missing")
	op := path.Get
	require.NotNil(t, op)

	return map[string]any{
		"modelTitle":           gadget.Title,
		"modelDescription":     gadget.Description,
		"operationSummary":     op.Summary,
		"operationDescription": op.Description,
	}
}

// goldenJSON compares the fragment to (or, under UPDATE_GOLDEN, rewrites)
// testdata/<feature>.json.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenJSON(t *testing.T, frag map[string]any, feature string) {
	t.Helper()
	got, err := json.MarshalIndent(frag, "", "  ")
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

// TestSingleLineCommentAsDescription emits the off/on goldens and asserts the
// toggle: a single-line comment is a title/summary by default, a description
// when the option is on.
func TestSingleLineCommentAsDescription(t *testing.T) {
	off := fragment(t, scan(t, false))
	on := fragment(t, scan(t, true))

	goldenJSON(t, off, "off")
	goldenJSON(t, on, "on")

	// Default: the single-line comment is the title (model) / summary (op).
	assert.Equal(t, "Gadget is a small device.", off["modelTitle"])
	assert.Empty(t, off["modelDescription"])
	assert.Equal(t, "Lists every gadget in the catalog.", off["operationSummary"])
	assert.Empty(t, off["operationDescription"])

	// On: the same comment is routed to the description; no title / summary.
	assert.Empty(t, on["modelTitle"])
	assert.Equal(t, "Gadget is a small device.", on["modelDescription"])
	assert.Empty(t, on["operationSummary"])
	assert.Equal(t, "Lists every gadget in the catalog.", on["operationDescription"])
}
