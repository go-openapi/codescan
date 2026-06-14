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

// TestCoverage_Bug1133 locks the fix for go-swagger issues #1133 and #1174 ("scan
// models chokes on unsupported type"): an unsupported (function) type in the
// scanned package — even one used by a non-annotated struct — no longer halts the
// whole scan. It is warned and skipped; the annotated model is still emitted.
func TestCoverage_Bug1133(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1133/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err, "an unsupported func type must not halt the scan")

	assert.Contains(t, doc.Definitions, "foo", "the annotated model is emitted")
	assert.NotContains(t, doc.Definitions, "TranslateFunc", "the func type is skipped")
	assert.NotContains(t, doc.Definitions, "SomeStruct", "the non-annotated user struct is not emitted")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1133_schema.json")
}
