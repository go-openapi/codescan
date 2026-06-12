// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_DiagnosticsDeduped pins the OnDiagnostic dedup: the build
// re-processes the same field/annotation across passes (most visibly a
// swagger:parameters struct applied to several operation ids, which rebuilds
// every field once per id), but the callback stream must surface each exact
// diagnostic — same position, code and message — at most once. The
// classification corpus exercises exactly that multi-operation pattern.
func TestCoverage_DiagnosticsDeduped(t *testing.T) {
	counts := map[string]int{}
	emitted := 0
	_, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./goparsing/classification/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			counts[d.Pos.String()+"|"+string(d.Code)+"|"+d.Message]++
			emitted++
		},
	})
	require.NoError(t, err)
	require.Positive(t, emitted, "the classification fixture should surface diagnostics")

	for key, c := range counts {
		assert.Equal(t, 1, c, "diagnostic emitted %d times (must be deduped): %s", c, key)
	}
	assert.Equal(t, len(counts), emitted, "no diagnostic should be emitted more than once")
}
