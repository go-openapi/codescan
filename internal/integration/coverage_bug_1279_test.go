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

// TestCoverage_Bug1279 locks the answer to go-swagger issue #1279 ("add path
// parameters to swagger:route without using structs"): an inline `Parameters:`
// block inside the swagger:route comment declares parameters directly — no
// wrapper swagger:parameters struct required.
//
// 📖 Need doc: document the inline Parameters: block in swagger:route.
func TestCoverage_Bug1279(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1279/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/{country}"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	p := op.Parameters[0]
	assert.Equal(t, "country", p.Name)
	assert.Equal(t, "path", p.In)
	assert.Equal(t, "string", p.Type)
	assert.True(t, p.Required)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1279_schema.json")
}
