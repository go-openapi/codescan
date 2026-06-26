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

// TestCoverage_Bug2783 locks the fix for go-swagger issue #2783: two packages each declare
// `swagger:model Test`.
//
// They used to collide on the short key "Test" and SILENTLY MERGE (union of both structs' fields,
// last-package-wins, non-deterministic).
// Now each keeps its own identity: two distinct Test definitions (deep-keyed by package while the
// leaf collides) plus TestResponseBody — three definitions, deterministic.
//
// See the name-identity / cyclic-$ref design (.claude/plans/name-identity-cyclic-ref.md).
func TestCoverage_Bug2783(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2783/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	require.Len(t, doc.Definitions, 3,
		"TestResponseBody + two distinct Tests (b.Test and c.Test no longer merge)")
	assert.NotContains(t, doc.Definitions, "Test", "no merged bare Test key")

	// The two Tests are distinct, qualified by their package leaf: BTest carries only b's field, CTest
	// only c's.
	bTest, okB := doc.Definitions["BTest"]
	cTest, okC := doc.Definitions["CTest"]
	require.True(t, okB, "b.Test -> BTest")
	require.True(t, okC, "c.Test -> CTest")
	assert.Contains(t, bTest.Properties, "b")
	assert.NotContains(t, bTest.Properties, "c", "BTest must not absorb c's field")
	assert.Contains(t, cTest.Properties, "c")
	assert.NotContains(t, cTest.Properties, "b", "CTest must not absorb b's field")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2783_schema.json")
}
