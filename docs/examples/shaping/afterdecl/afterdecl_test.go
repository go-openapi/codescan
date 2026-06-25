// SPDX-License-Identifier: Apache-2.0

package afterdecl

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

// scanAfterDecl scans the witness package with the AfterDeclComments opt-in
// either off (the control) or on.
func scanAfterDecl(t *testing.T, on bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:           examplesRoot(t),
		Packages:          []string{"./shaping/afterdecl"},
		ScanModels:        true,
		AfterDeclComments: on,
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

// TestAfterDecl pins the opt-in's effect: off, the inside-body / inlined
// annotations are inert and nothing is discovered; on, the same source yields
// the three definitions, with their keywords applied.
func TestAfterDecl(t *testing.T) {
	// Off (control): the godoc above each decl is clean, so nothing is found.
	off := scanAfterDecl(t, false)
	goldenRaw(t, "off", off.Definitions)
	assert.Empty(t, off.Definitions, "with the option off, the annotations are inert")

	// On: the inside-body and inlined trailing annotations are discovered.
	on := scanAfterDecl(t, true)
	goldenRaw(t, "on", on.Definitions)

	// struct inside-body: swagger:model widgetModel + maxProperties: 5.
	widget, ok := on.Definitions["widgetModel"]
	require.True(t, ok, "widgetModel definition missing")
	require.NotNil(t, widget.MaxProperties)
	assert.Equal(t, int64(5), *widget.MaxProperties)

	// field trailing comment: swagger:strfmt date.
	created, ok := widget.Properties["created"]
	require.True(t, ok, "created property missing")
	assert.Equal(t, "date", created.Format)

	// defined type trailing: swagger:model countType → integer.
	count, ok := on.Definitions["countType"]
	require.True(t, ok, "countType definition missing")
	assert.Contains(t, count.Type, "integer")

	// type alias trailing: swagger:model stampType → string.
	stamp, ok := on.Definitions["stampType"]
	require.True(t, ok, "stampType definition missing")
	assert.Contains(t, stamp.Type, "string")

	// The route in the func body is discovered either way (location-agnostic).
	require.NotNil(t, on.Paths)
	_, ok = on.Paths.Paths["/widgets"]
	assert.True(t, ok, "the /widgets route is discovered")
}
