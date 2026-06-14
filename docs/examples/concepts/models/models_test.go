// SPDX-License-Identifier: Apache-2.0

package models

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

// examplesRoot returns the absolute path to the examples module root, so the
// scan resolves "./concepts/models" regardless of the test's working directory.
func examplesRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// thisFile is <root>/concepts/models/models_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func scanModels(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./concepts/models"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenFor marshals one definition and compares it to (or, under
// UPDATE_GOLDEN, rewrites) testdata/<feature>.json — the fragment the
// "Model definitions" tutorial renders next to the annotated source. Keeping
// the panes golden-checked means the documentation can never drift from what
// the scanner actually emits.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenFor(t *testing.T, doc *spec.Swagger, feature, defName string) {
	t.Helper()
	schema, ok := doc.Definitions[defName]
	require.Truef(t, ok, "definition %q not found (have %v)", defName, definitionNames(doc))

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

func definitionNames(doc *spec.Swagger) []string {
	names := make([]string, 0, len(doc.Definitions))
	for k := range doc.Definitions {
		names = append(names, k)
	}
	return names
}

// TestModelFragments emits and verifies the per-annotation golden fragments the
// tutorial pairs with each source region.
func TestModelFragments(t *testing.T) {
	doc := scanModels(t)

	cases := []struct{ feature, defName string }{
		{"model", "Pet"},        // swagger:model — a plain struct definition
		{"strfmt", "Device"},    // swagger:strfmt — a {type:string, format} field
		{"enum", "Task"},        // swagger:enum — enum values inlined on the field
		{"allof", "Dog"},        // swagger:allOf — two base $refs + inline arm
		{"type", "Token"},       // swagger:type — overridden field type (type-level)
		{"typefield", "Coupon"}, // swagger:type — overridden type on the field doc
		{"name", "Car"},         // swagger:name — overridden interface-method property
	}
	for _, tc := range cases {
		t.Run(tc.feature, func(t *testing.T) {
			goldenFor(t, doc, tc.feature, tc.defName)
		})
	}
}

// TestIgnoreOmitsType documents swagger:ignore: the Secret type is scanned,
// classified, then dropped — so it never reaches the definitions.
func TestIgnoreOmitsType(t *testing.T) {
	doc := scanModels(t)
	_, found := doc.Definitions["Secret"]
	assert.False(t, found, "swagger:ignore type Secret must not appear in definitions")
}
