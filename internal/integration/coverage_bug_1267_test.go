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

// TestCoverage_Bug1267 locks go-swagger issue #1267 ("response files?"): a
// file-download response is described with produces: application/octet-stream and
// a body field marked swagger:strfmt binary -> {type:string, format:binary}.
func TestCoverage_Bug1267(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1267/..."}, WorkDir: scantest.FixturesDir(),
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"application/octet-stream"}, doc.Paths.Paths["/excel"].Get.Produces)
	sch := doc.Responses["binary"].Schema
	require.NotNil(t, sch)
	assert.Equal(t, []string{"string"}, []string(sch.Type))
	assert.Equal(t, "binary", sch.Format, "binary body schema (go-swagger#1267)")
	scantest.CompareOrDumpJSON(t, doc, "bugs_1267_schema.json")
}
