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

// TestCoverage_Bug1026 locks the resolution to go-swagger issue #1026
// ("suggestion: x-logo feature"): no dedicated knob is needed — a custom vendor
// extension (x-logo) is declared in the swagger:meta block via InfoExtensions
// and lands on the spec's info object.
//
// 📖 Need doc: document InfoExtensions / Extensions for arbitrary x-* vendor
// extensions on info / the spec root.
func TestCoverage_Bug1026(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1026/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc.Info)

	logo, ok := doc.Info.Extensions["x-logo"]
	require.True(t, ok, "x-logo is emitted as an info vendor extension")
	logoMap, ok := logo.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "./images/logo.png", logoMap["url"])

	scantest.CompareOrDumpJSON(t, doc, "bugs_1026_schema.json")
}
