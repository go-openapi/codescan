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

// TestCoverage_Bug1436 locks go-swagger issue #1436 (the -m semantics): without --scan-models, only
// route-reachable models are emitted; with -m, every swagger:model is emitted (including standalone
// ones).
//
// Works-as-designed; the "prune unreferenced under -m" middle ground is forthcoming-features §12.
func TestCoverage_Bug1436(t *testing.T) {
	run := func(m bool) map[string]struct{} {
		doc, err := codescan.Run(&codescan.Options{
			Packages: []string{"./bugs/1436/..."}, WorkDir: scantest.FixturesDir(), ScanModels: m,
		})
		require.NoError(t, err)
		got := map[string]struct{}{}
		for k := range doc.Definitions {
			got[k] = struct{}{}
		}
		return got
	}
	nom := run(false)
	_, reachable := nom["Reachable"]
	_, standaloneNoM := nom["Standalone"]
	assert.True(t, reachable, "route-reachable model is always emitted")
	assert.False(t, standaloneNoM, "standalone swagger:model is NOT emitted without -m (go-swagger#1436)")

	withM := run(true)
	_, standaloneM := withM["Standalone"]
	assert.True(t, standaloneM, "standalone swagger:model IS emitted with -m")
}
