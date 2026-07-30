// SPDX-License-Identifier: Apache-2.0

package nested

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

// scanNested scans the hierarchy WITHOUT ScanModels: only the root is referenced (by the route's
// response), so every level below it has to be discovered.
func scanNested(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:  examplesRoot(t),
		Packages: []string{"./concepts/polymorphism-nested"},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	return doc
}

// goldenFor emits and verifies one definition fragment, rewriting it under UPDATE_GOLDEN.
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

// TestNestedHierarchy backs the "Multi-level hierarchies" section: a route references only the root,
// and the levels below it are discovered in cascade — the root pulls the intermediate type, which in
// turn pulls the leaf.
//
// It also locks where each level's discriminator lands, which is the part that surprises: the root
// carries it at the top level, while the intermediate type — being itself composed with allOf —
// carries it inside its own allOf member.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestNestedHierarchy(t *testing.T) {
	doc := scanNested(t)

	for _, name := range []string{"Shape", "Polygon", "Square"} {
		assert.Containsf(t, doc.Definitions, name, "%s must be emitted without ScanModels", name)
	}
	assert.Len(t, doc.Definitions, 3)

	root := doc.Definitions["Shape"]
	assert.Equal(t, "shapeType", root.Discriminator, "the root carries its discriminator at the top level")

	mid := doc.Definitions["Polygon"]
	require.Len(t, mid.AllOf, 2, "the intermediate type is itself a subtype: allOf[root, own]")
	assert.Equal(t, "#/definitions/Shape", mid.AllOf[0].Ref.String())
	assert.Empty(t, mid.Discriminator, "an intermediate type has no top-level discriminator")
	assert.Equal(t, "polygonType", mid.AllOf[1].Discriminator,
		"it declares its own discriminator inside its own allOf member")

	leaf := doc.Definitions["Square"]
	require.Len(t, leaf.AllOf, 2)
	assert.Equal(t, "#/definitions/Polygon", leaf.AllOf[0].Ref.String(),
		"a leaf refers to the level directly above it, not to the root")

	goldenFor(t, doc, "root", "Shape")           // discriminator at the top level
	goldenFor(t, doc, "intermediate", "Polygon") // subtype AND base
	goldenFor(t, doc, "leaf", "Square")          // allOf: [$ref Polygon, own fields]
}
