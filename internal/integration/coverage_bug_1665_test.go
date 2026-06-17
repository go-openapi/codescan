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

// TestCoverage_Bug1665 locks the resolution to go-swagger issue #1665 ("can't
// combine a same annotation across multiple lines"): multiple swagger:parameters
// lines on one type are all honored — the shared param binds to operations listed
// across every line.
func TestCoverage_Bug1665(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1665/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	hasID := func(path string) bool {
		op := doc.Paths.Paths[path].Get
		if op == nil {
			return false
		}
		for _, p := range op.Parameters {
			if p.Name == "id" && p.In == "path" {
				return true
			}
		}
		return false
	}
	assert.True(t, hasID("/foo/{id}"), "op from the first swagger:parameters line gets the param")
	assert.True(t, hasID("/bar/{id}"), "op from the second swagger:parameters line gets the param")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1665_schema.json")
}
