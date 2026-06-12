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

// TestCoverage_Bug3107 locks the fix for go-swagger issue #3107 ("No
// struct definition in swagger generate"). Under `generate spec -m`
// (ScanModels), a swagger:model struct must be emitted with its full
// shape — `type: object` plus its properties — rather than a bare
// definition carrying only `x-go-package` with the fields dropped.
//
// The fixture places the model in its own package, referenced
// cross-package by a route's response body, so the test also exercises
// type resolution through the import graph (the v0.30.5 trigger: the
// model package was resolved by reference, not directly scanned).
func TestCoverage_Bug3107(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3107/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	def, ok := doc.Definitions["MyStruct"]
	require.True(t, ok, "MyStruct must be emitted as a definition")

	// The regression: an empty stub with only x-go-package.
	require.NotEmpty(t, def.Type, "definition must declare its type, not be an empty stub")
	assert.Equal(t, "object", def.Type[0])
	require.Len(t, def.Properties, 2, "both struct fields must be present")
	assert.Contains(t, def.Properties, "field1")
	assert.Contains(t, def.Properties, "field2")
	assert.Equal(t, "string", def.Properties["field1"].Type[0])
	assert.Equal(t, "string", def.Properties["field2"].Type[0])

	scantest.CompareOrDumpJSON(t, doc, "bugs_3107_schema.json")
}
