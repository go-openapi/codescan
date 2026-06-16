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

// TestCoverage_Bug91 locks go-swagger issue #91 ("decouple from source"): a query
// parameter can be declared INLINE in the swagger:operation YAML body — no Go
// wrapper struct required.
func TestCoverage_Bug91(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{Packages: []string{"./bugs/91/..."}, WorkDir: scantest.FixturesDir()})
	require.NoError(t, err)
	op := doc.Paths.Paths["/search"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	assert.Equal(t, "type", op.Parameters[0].Name)
	assert.Equal(t, "query", op.Parameters[0].In)
	scantest.CompareOrDumpJSON(t, doc, "bugs_91_schema.json")
}
