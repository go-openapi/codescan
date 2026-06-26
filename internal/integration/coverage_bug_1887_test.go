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

// TestCoverage_Bug1887 locks go-swagger issue #1887 ("file type support"): the `swagger:file`
// marker on a formData parameter field emits the Swagger 2.0 `type: file` parameter.
func TestCoverage_Bug1887(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1887/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/upload"].Post
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	p := op.Parameters[0]
	assert.Equal(t, "upfile", p.Name)
	assert.Equal(t, "formData", p.In)
	assert.Equal(t, "file", p.Type)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1887_schema.json")
}
