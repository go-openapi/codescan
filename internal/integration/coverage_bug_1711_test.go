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

// TestCoverage_Bug1711 documents the resolution to go-swagger issue #1711 ("how to add a different
// description to parameters"): a parameter's description is the comment text, kept distinct from
// the Go field name (carried separately as the x-go-name vendor extension).
//
// 📖 Need doc: the param description comes from the field comment; x-go-name is a separate vendor
// extension, not concatenated into the description.
func TestCoverage_Bug1711(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1711/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/things"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	p := op.Parameters[0]
	assert.Equal(t, "the maximum number of items to return", p.Description)
	assert.Equal(t, "Limit", p.Extensions["x-go-name"])

	scantest.CompareOrDumpJSON(t, doc, "bugs_1711_schema.json")
}
