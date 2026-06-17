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

// TestCoverage_Bug1118 documents (current behaviour) go-swagger issue #1118
// ("sometimes title, sometimes description"). This is the same title/description
// heuristic as #2626: a single-line model comment ending in a period becomes the
// title; without a trailing period it becomes the description; a multi-line
// comment splits into title (first line) + description (rest). Kept as-is — an
// opt-in knob to force description-only is tracked as forthcoming-features.md §13.
//
// 📖 Need doc: document the title/description heuristic (see #2626).
func TestCoverage_Bug1118(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1118/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	wp := doc.Definitions["WithPeriod"]
	assert.Equal(t, "WithPeriod is a thing.", wp.Title)
	assert.Empty(t, wp.Description, "trailing period -> title only")

	wo := doc.Definitions["WithoutPeriod"]
	assert.Empty(t, wo.Title, "no trailing period -> description only")
	assert.Equal(t, "WithoutPeriod is a thing", wo.Description)

	ml := doc.Definitions["MultiLine"]
	assert.Equal(t, "MultiLine is a thing.", ml.Title)
	assert.Contains(t, ml.Description, "longer description")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1118_schema.json")
}
