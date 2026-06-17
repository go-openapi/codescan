// SPDX-License-Identifier: Apache-2.0

package refoverride

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

func scanRefOverride(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./concepts/refoverride"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestRefOverride emits and verifies testdata/refoverride.json — the Person
// definition the "Decorating a $ref" subsection renders — and asserts the $ref
// is wrapped in an allOf so the field's description and x-* extension survive.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestRefOverride(t *testing.T) {
	doc := scanRefOverride(t)

	person, ok := doc.Definitions["Person"]
	require.True(t, ok, "Person definition missing")
	assert.Contains(t, person.Required, "home", "required rides the parent model")

	home := person.Properties["home"]
	require.Len(t, home.AllOf, 1, "the $ref is wrapped as an allOf member")
	assert.Equal(t, "#/definitions/Address", home.AllOf[0].Ref.String())
	assert.Equal(t, "Home is where the person lives.", home.Description, "description survives beside the $ref")
	assert.Contains(t, home.Extensions, "x-ui-order", "the vendor extension survives beside the $ref")

	got, err := json.MarshalIndent(person, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", "refoverride.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}
