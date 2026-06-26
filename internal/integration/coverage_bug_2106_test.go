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

// TestCoverage_Bug2106 locks go-swagger issue #2106 ("Extensions ignored on models when type is not
// an array"): a field-level `Extensions:` block emits its x-* vendor extensions on BOTH scalar and
// array fields.
func TestCoverage_Bug2106(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2106/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	props := doc.Definitions["User"].Properties
	assert.Equal(t, "scalarvalue", props["name"].Extensions["x-scalar-ext"],
		"extensions must emit on a scalar field (go-swagger#2106)")
	assert.Equal(t, "arrayvalue", props["friends"].Extensions["x-array-ext"],
		"extensions also emit on an array field")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2106_schema.json")
}
