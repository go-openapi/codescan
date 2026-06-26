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

// TestCoverage_Bug958 locks the fix for go-swagger issue #958 ("how to specify sample values in
// user structs").
//
// A field-level `default:` / `example:` was carried for a plain builtin field but dropped when the
// field used a named (defined) type.
// The fix carries both for defined-type fields too; since a $ref cannot carry sibling keywords,
// they ride the override arm of an allOf compound.
func TestCoverage_Bug958(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/958/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Plain builtin field: default/example sit inline on the property.
	plain := doc.Definitions["PlainStruct"]
	require.Contains(t, plain.Properties, "value")
	pv := plain.Properties["value"]
	assert.Equal(t, "string", pv.Type[0])
	assert.Equal(t, "samplesample", pv.Default)
	assert.Equal(t, "examplevalue", pv.Example)

	// Defined-type field: the regression. $ref + override arm carrying both default and example.
	defined := doc.Definitions["DefinedStruct"]
	require.Contains(t, defined.Properties, "value")
	dv := defined.Properties["value"]
	assert.Empty(t, dv.Ref.String(), "the $ref must not sit beside the override keywords")
	require.Len(t, dv.AllOf, 2, "defined-type field with overrides → two-arm allOf")
	assert.Equal(t, "#/definitions/SpecificType", dv.AllOf[0].Ref.String())
	assert.Equal(t, "samplesample", dv.AllOf[1].Default, "default must be carried for the defined type")
	assert.Equal(t, "examplevalue", dv.AllOf[1].Example, "example must be carried for the defined type")

	scantest.CompareOrDumpJSON(t, doc, "bugs_958_schema.json")
}
