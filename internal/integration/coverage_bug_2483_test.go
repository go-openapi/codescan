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

// TestCoverage_Bug2483 locks the fix for go-swagger issue #2483 ("generation
// from code not working with allOf"): a model embedding a swagger:allOf base
// plus a plain inline struct used to duplicate the allOf arms (the $ref and the
// inline object each appeared twice), which tripped a "circular ancestry"
// validation error. The arms are now emitted exactly once.
func TestCoverage_Bug2483(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2483/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	pet := doc.Definitions["Pet"]
	require.Len(t, pet.AllOf, 2, "exactly two allOf arms — no duplication (the #2483 bug)")
	assert.Equal(t, "#/definitions/SimpleOne", pet.AllOf[0].Ref.String())

	inline := pet.AllOf[1]
	assert.Empty(t, inline.Ref.String(), "the second arm is the inline merge, not another $ref")
	for _, prop := range []string{"did", "cat", "notes", "extra"} {
		assert.Contains(t, inline.Properties, prop)
	}

	scantest.CompareOrDumpJSON(t, doc, "bugs_2483_schema.json")
}
