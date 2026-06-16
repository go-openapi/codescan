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

// TestCoverage_Bug2871 locks the reasonable answer to go-swagger issue #2871: the
// literal "dynamic examples" framing is out of scope, but distinct per-operation,
// per-response examples (by mime type) ARE supported — multiple operations may
// reuse one example-bearing model while each response code carries its own
// example, different from the model's.
func TestCoverage_Bug2871(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2871/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true,
	})
	require.NoError(t, err)

	get := doc.Paths.Paths["/widgets/{id}"].Get.Responses.StatusCodeResponses
	post := doc.Paths.Paths["/widgets"].Post.Responses.StatusCodeResponses
	assert.Equal(t, map[string]any{"name": "alice-200"}, get[200].Examples["application/json"])
	assert.Equal(t, map[string]any{"name": "not-found-404"}, get[404].Examples["application/json"])
	assert.Equal(t, map[string]any{"name": "created-201"}, post[201].Examples["application/json"])
	assert.Equal(t, map[string]any{"name": "conflict-409"}, post[409].Examples["application/json"])
	// the shared model keeps its own distinct example
	assert.Equal(t, "model-default", doc.Definitions["Widget"].Properties["name"].Example)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2871_schema.json")
}
