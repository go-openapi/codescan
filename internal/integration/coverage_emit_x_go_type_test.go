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

// TestCoverage_EmitXGoType locks the answer to go-swagger issue #2924 ("emit
// x-go-type vendor extension"): the opt-in EmitXGoType option stamps an
// x-go-type extension on every emitted definition (the fully-qualified Go
// type), default-off, and suppressed under the SkipExtensions umbrella.
func TestCoverage_EmitXGoType(t *testing.T) {
	const pkgType = "github.com/go-openapi/codescan/fixtures/enhancements/emit-x-go-type.Widget"

	// Default: x-go-name / x-go-package are present, x-go-type is not.
	off, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/emit-x-go-type/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true,
	})
	require.NoError(t, err)
	w := off.Definitions["Widget"]
	assert.Equal(t, "github.com/go-openapi/codescan/fixtures/enhancements/emit-x-go-type", w.Extensions["x-go-package"])
	assert.Nil(t, w.Extensions["x-go-type"], "x-go-type is opt-in (off by default)")

	// EmitXGoType: x-go-type records the fully-qualified Go type.
	on, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/emit-x-go-type/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true,
		EmitXGoType: true,
	})
	require.NoError(t, err)
	assert.Equal(t, pkgType, on.Definitions["Widget"].Extensions["x-go-type"], "EmitXGoType stamps the qualified Go type (go-swagger#2924)")

	// SkipExtensions wins: no vendor extension is emitted even with EmitXGoType.
	none, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/emit-x-go-type/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true,
		EmitXGoType: true, SkipExtensions: true,
	})
	require.NoError(t, err)
	nw := none.Definitions["Widget"]
	assert.Nil(t, nw.Extensions["x-go-type"], "SkipExtensions suppresses x-go-type")
	assert.Nil(t, nw.Extensions["x-go-package"], "SkipExtensions suppresses x-go-package")
}
