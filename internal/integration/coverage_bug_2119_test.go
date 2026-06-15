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

// TestCoverage_Bug2119 locks the answer to go-swagger issue #2119 ("add flag to
// skip generation of x-go-name"): the `SkipExtensions` option suppresses the
// scanner-derived x-go-name / x-go-package vendor extensions (which clash across
// same-named types).
func TestCoverage_Bug2119(t *testing.T) {
	// Default: x-go-name / x-go-package are present.
	on, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2119/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "Colour", on.Definitions["Widget"].Properties["colour"].Extensions["x-go-name"])

	// SkipExtensions: x-go-* are gone.
	off, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2119/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true,
		SkipExtensions: true,
	})
	require.NoError(t, err)
	w := off.Definitions["Widget"]
	assert.Nil(t, w.Extensions["x-go-package"], "SkipExtensions drops x-go-package (go-swagger#2119)")
	assert.Nil(t, w.Properties["colour"].Extensions["x-go-name"], "SkipExtensions drops x-go-name")

	scantest.CompareOrDumpJSON(t, off, "bugs_2119_schema.json")
}
