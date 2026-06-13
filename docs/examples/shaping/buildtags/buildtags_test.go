// SPDX-License-Identifier: Apache-2.0

package buildtags

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

func scan(t *testing.T, tags string) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/buildtags"},
		ScanModels: true,
		BuildTags:  tags,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenDefs marshals the whole definitions map and compares it to (or, under
// UPDATE_GOLDEN, rewrites) testdata/<feature>.json.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenDefs(t *testing.T, doc *spec.Swagger, feature string) {
	t.Helper()
	got, err := json.MarshalIndent(doc.Definitions, "", "  ")
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

// TestBuildTags emits the before/after fragments and asserts the toggle: the
// tag-gated Experimental type is scanned only when BuildTags includes its tag.
func TestBuildTags(t *testing.T) {
	off := scan(t, "")
	on := scan(t, "experimental")

	goldenDefs(t, off, "off")
	goldenDefs(t, on, "on")

	_, offHas := off.Definitions["Experimental"]
	assert.False(t, offHas, "default: tag-gated Experimental must not be scanned")
	_, onHas := on.Definitions["Experimental"]
	assert.True(t, onHas, "with BuildTags=experimental: Experimental must be scanned")
}
