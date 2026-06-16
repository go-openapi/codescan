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

// TestCoverage_Bug2184 locks go-swagger issue #2184 ("spec generation fails to
// replace $ref with indicated type when using swagger:type on struct"): a struct
// type carrying `swagger:type int64`, used as a parameter field, makes the
// parameter an integer — not a $ref to the struct.
func TestCoverage_Bug2184(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2184/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	p := doc.Paths.Paths["/object/{id}"].Get.Parameters[0]
	assert.Equal(t, "id", p.Name)
	assert.Equal(t, "integer", p.Type, "swagger:type on the struct overrides the param to integer (go-swagger#2184)")
	assert.Equal(t, "int64", p.Format)
	assert.Empty(t, p.Schema, "no $ref/schema to the overridden struct")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2184_schema.json")
}
