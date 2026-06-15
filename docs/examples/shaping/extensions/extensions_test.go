// SPDX-License-Identifier: Apache-2.0

package extensions

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

func scan(t *testing.T, skipExtensions bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:        examplesRoot(t),
		Packages:       []string{"./shaping/extensions"},
		ScanModels:     true,
		SkipExtensions: skipExtensions,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenDef marshals the Widget definition and compares it to (or, under
// UPDATE_GOLDEN, rewrites) testdata/<feature>.json.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenDef(t *testing.T, doc *spec.Swagger, feature string) string {
	t.Helper()
	schema, ok := doc.Definitions["Widget"]
	require.True(t, ok, "Widget definition missing")
	got, err := json.MarshalIndent(schema, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", feature+".json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
	return string(got)
}

// TestSkipExtensions emits the before/after fragments and asserts the toggle:
// x-go-* extensions are present by default and gone when SkipExtensions is on.
func TestSkipExtensions(t *testing.T) {
	withExt := goldenDef(t, scan(t, false), "off") // default: extensions present
	noExt := goldenDef(t, scan(t, true), "on")     // SkipExtensions: gone

	assert.True(t, strings.Contains(withExt, "x-go-name"), "default output should carry x-go-* extensions")
	assert.False(t, strings.Contains(noExt, "x-go-"), "SkipExtensions output must carry no x-go-* extensions")
}

// TestParamHeaderExtensions emits the golden showing author-supplied x-*
// extensions surviving on a parameter and a response header — and asserts they
// are present (and, being author-authored, survive SkipExtensions).
func TestParamHeaderExtensions(t *testing.T) {
	doc := scan(t, true) // SkipExtensions on: author x-* must still survive
	require.NotNil(t, doc.Paths)

	widgets, ok := doc.Paths.Paths["/widgets"]
	require.True(t, ok, "GET /widgets missing")
	params := widgets.Get.Parameters
	require.Len(t, params, 1)
	assert.Contains(t, params[0].Extensions, "x-example", "x-example survives on the parameter")

	resp, ok := doc.Responses["widgetList"]
	require.True(t, ok, "widgetList response missing")
	hdr := resp.Headers["X-Rate-Limit"]
	assert.Contains(t, hdr.Extensions, "x-units", "x-units survives on the response header")

	frag := map[string]any{"parameter": params[0], "responseHeader": hdr}
	got, err := json.MarshalIndent(frag, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", "paramext.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}
