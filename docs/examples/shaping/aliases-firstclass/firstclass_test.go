// SPDX-License-Identifier: Apache-2.0

package firstclass

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

// scanMode scans the package under one alias mode.
func scanMode(t *testing.T, opts codescan.Options) spec.Definitions {
	t.Helper()
	opts.WorkDir = examplesRoot(t)
	opts.Packages = []string{"./shaping/aliases-firstclass"}
	opts.ScanModels = true

	doc, err := codescan.Run(&opts)
	require.NoError(t, err)
	require.NotNil(t, doc)

	return doc.Definitions
}

// goldenDefs emits and verifies the whole definitions map for one mode.
func goldenDefs(t *testing.T, feature string, defs spec.Definitions) {
	t.Helper()
	got, err := json.MarshalIndent(defs, "", "  ")
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

// refOf returns the $ref carried by a property, or "" when it carries none.
func refOf(defs spec.Definitions, def, prop string) string {
	prd := defs[def].Properties[prop]

	return prd.Ref.String()
}

// TestFirstClassAlias_Expand locks the default: the alias definition is a
// structural COPY of its target, and use sites point at the alias.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestFirstClassAlias_Expand(t *testing.T) {
	defs := scanMode(t, codescan.Options{})

	require.Contains(t, defs, "Fee")
	assert.Contains(t, defs["Fee"].Properties, "cents",
		"the default expands the alias into a copy of the target")
	feeDefault := defs["Fee"]
	assert.Empty(t, feeDefault.Ref.String(), "no $ref chain in the default mode")
	assert.Equal(t, "#/definitions/Fee", refOf(defs, "Receipt", "charge"),
		"the use site keeps the alias name")

	goldenDefs(t, "expand", defs)
}

// TestFirstClassAlias_RefAliases locks RefAliases: the alias definition becomes a
// $ref chain to its target rather than a copy — one shape, two names.
func TestFirstClassAlias_RefAliases(t *testing.T) {
	defs := scanMode(t, codescan.Options{RefAliases: true})

	fee := defs["Fee"]
	assert.Equal(t, "#/definitions/Amount", fee.Ref.String(),
		"the alias definition is a $ref chain to the target")
	assert.Empty(t, defs["Fee"].Properties, "no duplicated properties under RefAliases")
	assert.Equal(t, "#/definitions/Fee", refOf(defs, "Receipt", "charge"),
		"the use site still keeps the alias name")

	goldenDefs(t, "refaliases", defs)
}

// TestFirstClassAlias_Transparent locks the part that surprises: TransparentAliases
// dissolves the alias at USE SITES, but a swagger:model-annotated alias is still
// published as its own definition under ScanModels — it just stops being
// referenced by anything.
func TestFirstClassAlias_Transparent(t *testing.T) {
	defs := scanMode(t, codescan.Options{TransparentAliases: true})

	assert.Equal(t, "#/definitions/Amount", refOf(defs, "Receipt", "charge"),
		"the use site dissolves to the target")
	require.Contains(t, defs, "Fee",
		"the annotated alias is still published; the option governs use sites, not the annotation")
	assert.Contains(t, defs["Fee"].Properties, "cents")

	// Nothing references Fee any more — with PruneUnusedModels it would be dropped.
	for _, def := range defs {
		for name, prop := range def.Properties {
			assert.NotEqualf(t, "#/definitions/Fee", prop.Ref.String(),
				"property %q still points at the dissolved alias", name)
		}
	}

	goldenDefs(t, "transparent", defs)
}
