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

// TestCoverage_Bug2985 verifies the fix for go-swagger issue #2985: minProperties / maxProperties
// (the reporter's "minAttributes/maxAttributes") are now grammar keywords (mirroring minItems /
// maxItems), so on an object swagger:model they emit property-count validation instead of leaking
// into the model description.
func TestCoverage_Bug2985(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2985/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	sch, ok := doc.Definitions["MyObjectType"]
	require.True(t, ok)

	// FIXED: property-count validation is emitted on the object schema.
	require.NotNil(t, sch.MinProperties)
	require.NotNil(t, sch.MaxProperties)
	assert.Equal(t, int64(1), *sch.MinProperties)
	assert.Equal(t, int64(10), *sch.MaxProperties)
	// FIXED: the keywords no longer leak into the description.
	assert.NotContains(t, sch.Description, "minProperties:")
	assert.NotContains(t, sch.Description, "maxProperties:")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2985_schema.json")
}
