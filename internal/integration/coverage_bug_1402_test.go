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

// TestCoverage_Bug1402 locks go-swagger issue #1402 ("additionalProperties breaks
// map[string]interface{}"): the field renders as {type:object, additionalProperties:{}} — an OPEN
// value schema (any), not one wrongly constrained to object.
func TestCoverage_Bug1402(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1402/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true,
	})
	require.NoError(t, err)
	data := doc.Definitions["Bag"].Properties["data"]
	assert.Equal(t, []string{"object"}, []string(data.Type))
	require.NotNil(t, data.AdditionalProperties)
	require.NotNil(t, data.AdditionalProperties.Schema)
	assert.Empty(t, data.AdditionalProperties.Schema.Type, "interface{} values are unconstrained (go-swagger#1402)")
	scantest.CompareOrDumpJSON(t, doc, "bugs_1402_schema.json")
}
