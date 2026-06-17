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

// TestCoverage_Bug1395 locks go-swagger issue #1395: the OP's tab-indented
// swagger:meta with a Security requirement + SecurityDefinitions now parses
// cleanly into a correct security requirement and apiKey definition. The
// original report (corrupted definitions) was a misclassified security entry,
// resolved by the meta-Security parsing fixes (cf. #2403). The OP's
// "SecurityDefinition" (singular) was a keyword typo.
func TestCoverage_Bug1395(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1395/..."}, WorkDir: scantest.FixturesDir(),
	})
	require.NoError(t, err)

	require.NotNil(t, doc.Info)
	assert.Equal(t, "1.0.0", doc.Info.Version)
	require.Len(t, doc.Security, 1)
	_, hasReq := doc.Security[0]["APIKey"]
	assert.True(t, hasReq, "the Security requirement names APIKey (go-swagger#1395)")
	def, ok := doc.SecurityDefinitions["APIKey"]
	require.True(t, ok, "the apiKey securityDefinition is emitted")
	assert.Equal(t, "apiKey", def.Type)
	assert.Equal(t, "Authorization", def.Name)
	assert.Equal(t, "header", def.In)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1395_schema.json")
}
