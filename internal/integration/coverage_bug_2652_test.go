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

// TestCoverage_Bug2652 locks the fix for go-swagger issue #2652 ("how to add
// complex example for a swagger:model"): an `example:` on a field whose type is
// a named model (rendered as a $ref) used to be dropped. It is now preserved on
// the override arm of an allOf compound (the #3125 shape), so the $ref and the
// example coexist.
//
// Known residual (F8 / coercion family): a complex JSON example on a $ref'd
// field is carried as a raw string ("{...}"), not a parsed object — the
// override arm has no type to coerce against. Asserted as-is below.
func TestCoverage_Bug2652(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2652/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	sel := doc.Definitions["Selector"]
	// Scalar field: example inline.
	assert.Equal(t, "apple selector", sel.Properties["name"].Example)

	// $ref'd field: example preserved on the allOf override arm (the #2652 fix).
	pod := sel.Properties["podSelector"]
	assert.Empty(t, pod.Ref.String(), "the $ref must not sit beside the example")
	require.Len(t, pod.AllOf, 2)
	assert.Equal(t, "#/definitions/LabelSelector", pod.AllOf[0].Ref.String())
	assert.Equal(t, `{"matchLabels":{}}`, pod.AllOf[1].Example,
		"the example is preserved (F8 residual: carried as a raw string, not a parsed object)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2652_schema.json")
}
