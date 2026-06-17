// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestQuirk_RefExampleCoercion verifies the fix for quirk G3
// (doc-site-quirks.md): a JSON-literal example: / default: on a field
// whose type is a $ref is coerced into a structured value on the allOf
// override arm — matching the direct-field path — instead of riding the
// arm as a raw string.
func TestQuirk_RefExampleCoercion(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./quirks/ref-example-coercion/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	props := doc.Definitions["Holder"].Properties
	require.NotNil(t, props)

	// override arm = allOf[1] (allOf[0] is the $ref).
	armExample := func(field string) any {
		p, ok := props[field]
		require.True(t, ok, "property %q present", field)
		require.Len(t, p.AllOf, 2, "%q is an allOf compound", field)
		return p.AllOf[1].Example
	}
	armDefault := func(field string) any {
		p, ok := props[field]
		require.True(t, ok, "property %q present", field)
		require.Len(t, p.AllOf, 2, "%q is an allOf compound", field)
		return p.AllOf[1].Default
	}

	// JSON-object example coerces to a map, not a raw string.
	assert.Equal(t, map[string]any{"name": "Rex"}, armExample("pet"))

	// JSON-object default coerces likewise.
	assert.Equal(t, map[string]any{"name": "Default"}, armDefault("defaultPet"))

	// JSON-array example coerces to a slice.
	assert.Equal(t, []any{
		map[string]any{"value": "a"},
		map[string]any{"value": "b"},
	}, armExample("tags"))

	// a non-JSON scalar is left as a plain string (referenced type
	// unknown on the override arm).
	assert.Equal(t, "plain-scalar", armExample("plain"))

	scantest.CompareOrDumpJSON(t, doc, "quirk_ref_example_coercion.json")
}
