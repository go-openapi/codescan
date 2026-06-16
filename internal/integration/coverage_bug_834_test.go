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

// TestCoverage_Bug834 locks go-swagger issue #834 (the field-annotation half):
// pattern / max length / example on a field are applied as schema fields, not
// concatenated into the property description. (The non-json naming-tag ask is the
// separate forthcoming-features §7 / #1391 item.)
func TestCoverage_Bug834(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{Packages: []string{"./bugs/834/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true})
	require.NoError(t, err)
	code := doc.Definitions["Widget"].Properties["code"]
	assert.Equal(t, "the code", code.Description, "description is just the prose, not the annotations")
	assert.Equal(t, "^[A-Z]+$", code.Pattern)
	require.NotNil(t, code.MaxLength)
	assert.EqualValues(t, 10, *code.MaxLength)
	assert.Equal(t, "ABC", code.Example)
	scantest.CompareOrDumpJSON(t, doc, "bugs_834_schema.json")
}
