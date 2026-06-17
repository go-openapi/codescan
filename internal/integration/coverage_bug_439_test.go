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

// TestCoverage_Bug439 locks go-swagger issue #439 ("too deep scans / Expr
// unsupported for a schema"): an unrelated, unannotated type with fields the
// scanner can't model (chan, time.Time) is left untouched — only the discovered
// body type is emitted, with no "Expr unsupported" error.
func TestCoverage_Bug439(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{Packages: []string{"./bugs/439/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true})
	require.NoError(t, err, "unrelated declarations must not break the scan (go-swagger#439)")
	_, hasBody := doc.Definitions["BodyParamType"]
	_, hasUnrelated := doc.Definitions["Unrelated"]
	assert.True(t, hasBody, "the discovered body type is emitted")
	assert.False(t, hasUnrelated, "the unrelated type is not pulled in")
	scantest.CompareOrDumpJSON(t, doc, "bugs_439_schema.json")
}
