// SPDX-License-Identifier: Apache-2.0

package polymorphism

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

func scanPolymorphism(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./concepts/polymorphism"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

func goldenFor(t *testing.T, doc *spec.Swagger, feature, defName string) {
	t.Helper()
	schema, ok := doc.Definitions[defName]
	require.Truef(t, ok, "definition %q not found", defName)

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

// TestPolymorphismFragments emits and verifies the golden fragments the tutorial
// renders, and asserts the discriminator lands on the base and the subtypes
// compose via allOf.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestPolymorphismFragments(t *testing.T) {
	doc := scanPolymorphism(t)

	pet, ok := doc.Definitions["Pet"]
	require.True(t, ok, "Pet definition missing")
	assert.Equal(t, "petType", pet.Discriminator, "discriminator: true names the property on the base schema")
	assert.Contains(t, pet.Required, "petType", "the discriminator property must be required")

	cat, ok := doc.Definitions["Cat"]
	require.True(t, ok, "Cat definition missing")
	require.Len(t, cat.AllOf, 2, "subtype composes via allOf: [$ref base, inline]")
	assert.Equal(t, "#/definitions/Pet", cat.AllOf[0].Ref.String())

	goldenFor(t, doc, "base", "Pet")     // discriminator on the base
	goldenFor(t, doc, "subtype", "Cat")  // allOf: [$ref Pet, own fields]
	goldenFor(t, doc, "subtype2", "Dog") // a second subtype
}
