// SPDX-License-Identifier: Apache-2.0

package nullable

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

func scan(t *testing.T, xNullable bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:                 examplesRoot(t),
		Packages:                []string{"./shaping/nullable"},
		ScanModels:              true,
		SetXNullableForPointers: xNullable,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenDef marshals the Profile definition and compares it to (or, under
// UPDATE_GOLDEN, rewrites) testdata/<feature>.json.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenDef(t *testing.T, doc *spec.Swagger, feature string) {
	t.Helper()
	schema, ok := doc.Definitions["Profile"]
	require.True(t, ok, "Profile definition missing")
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
}

// TestNullablePointers emits the before/after fragments and asserts the toggle:
// pointer fields acquire x-nullable only when SetXNullableForPointers is on.
func TestNullablePointers(t *testing.T) {
	off := scan(t, false)
	on := scan(t, true)

	goldenDef(t, off, "off")
	goldenDef(t, on, "on")

	offNick := off.Definitions["Profile"].Properties["nickname"]
	_, hasOff := offNick.Extensions["x-nullable"]
	assert.False(t, hasOff, "default: pointer field must not carry x-nullable")

	onNick := on.Definitions["Profile"].Properties["nickname"]
	assert.Equal(t, true, onNick.Extensions["x-nullable"], "enabled: pointer field must carry x-nullable")
}
