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

// TestCoverage_Bug2441 locks go-swagger issue #2441 ("file upload how to describe
// in annotations"): a raw (non-multipart) binary body is described with a string
// field marked `swagger:strfmt binary` → format binary. (OpenAPI 2.0 `type: file`
// is formData-only; a raw body uses `{type: string, format: binary}`.)
func TestCoverage_Bug2441(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2441/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/rawb"].Post
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	p := op.Parameters[0]
	assert.Equal(t, "body", p.In)
	assert.Equal(t, "binary", p.Format, "swagger:strfmt binary yields format binary (go-swagger#2441)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2441_schema.json")
}
