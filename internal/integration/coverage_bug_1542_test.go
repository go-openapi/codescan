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

// TestCoverage_Bug1542 locks the resolution to go-swagger issue #1542 ("inserting examples in
// response schema"): an example on a scalar field is applied, and an example on a map field is
// applied as an object — provided the map example is written as valid JSON (`{"k":"v"}`).
//
// 📖 Need doc: complex (map/object) examples must be written as valid JSON on the example: line.
// (Comma-list / array coercion remains the §2.1 enhancement.)
func TestCoverage_Bug1542(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1542/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	props := doc.Definitions["Metadata"].Properties
	assert.Equal(t, "OpenEBS Volume", props["name"].Example)

	ann := props["annotations"].Example
	m, ok := ann.(map[string]any)
	require.True(t, ok, "a JSON object example is parsed into a map, not kept as a string")
	assert.Equal(t, "SomeString", m["com.example1.com"])

	scantest.CompareOrDumpJSON(t, doc, "bugs_1542_schema.json")
}
