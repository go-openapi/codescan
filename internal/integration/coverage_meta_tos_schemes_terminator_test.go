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

// TestCoverage_MetaTOSSchemesTerminator pins the sibling-terminator
// rule for `Terms Of Service:` → `Schemes:` (Q26).
//
// Q26's original observation (Schemes absorbed into TOS body) no
// longer reproduces on master — the fix landed during Stream M's
// merge sequence, point of resolution not isolated. This test serves
// as a regression-detector: if a future lexer change re-breaks the
// terminator-family overlap for asRawBlock siblings in the meta
// scope, this turns red.
//
// Expected shape:
//
//	"termsOfService": "http://example.com/terms/"
//	"schemes":        ["http", "https"]
func TestCoverage_MetaTOSSchemesTerminator(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/meta-tos-schemes-terminator/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Equal(t, "http://example.com/terms/", doc.Info.TermsOfService,
		"TOS body must terminate at the Schemes: sibling, no carryover")
	assert.ElementsMatch(t, []string{"http", "https"}, doc.Schemes,
		"Schemes must be its own keyword, not absorbed into TOS text")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_meta_tos_schemes_terminator.json")
}
