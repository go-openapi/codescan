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

// TestCoverage_Bug2133 locks go-swagger issue #2133 ("spec generating schema and
// type"): a path parameter is emitted as a simple {type:string, in:path} with NO
// sibling `schema` (the double-emission the reporter saw would fail validation).
func TestCoverage_Bug2133(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2133/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	p := doc.Paths.Paths["/accounts/{accounts}"].Get.Parameters[0]
	assert.Equal(t, "path", p.In)
	assert.Equal(t, "string", p.Type)
	assert.Nil(t, p.Schema, "a path param must not also carry a schema (go-swagger#2133)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2133_schema.json")
}
