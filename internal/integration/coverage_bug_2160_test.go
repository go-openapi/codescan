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

// TestCoverage_Bug2160 locks go-swagger issue #2160 ("example of array of structs is 0 valued"): an
// array-of-structs field accepts an inline JSON-array example and emits it as an array of objects
// (not zero-valued). (The multi-line YAML-list example syntax is kept as a raw string — the
// example-coercion gap tracked in forthcoming-features §2.1.)
func TestCoverage_Bug2160(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2160/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	fb := doc.Definitions["ResponseBody"].Properties["fooBars"]
	ex, ok := fb.Example.([]any)
	require.True(t, ok, "the array example is emitted as a list, not zero-valued (go-swagger#2160)")
	require.Len(t, ex, 2)
	first, ok := ex[0].(map[string]any)
	require.True(t, ok)
	assert.EqualValues(t, 1, first["amount"])

	scantest.CompareOrDumpJSON(t, doc, "bugs_2160_schema.json")
}
