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

// TestCoverage_Bug1483 locks go-swagger issue #1483 ("items doesn't support maps
// whilst using map[string]string"): a map[string]string field in a body
// parameter renders as {type:object, additionalProperties:{type:string}} — no
// "items doesn't support maps" error.
func TestCoverage_Bug1483(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1483/..."}, WorkDir: scantest.FixturesDir(),
	})
	require.NoError(t, err)
	body := doc.Paths.Paths["/snap"].Post.Parameters[0].Schema
	require.NotNil(t, body)
	labels := body.Properties["labels"]
	require.NotNil(t, labels.AdditionalProperties)
	require.NotNil(t, labels.AdditionalProperties.Schema)
	assert.Equal(t, []string{"string"}, []string(labels.AdditionalProperties.Schema.Type))
	scantest.CompareOrDumpJSON(t, doc, "bugs_1483_schema.json")
}
