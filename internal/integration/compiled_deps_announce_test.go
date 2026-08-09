// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package integration_test

import (
	"os"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Taking dependency types from compiled export data is what an ordinary scan now does, so it is not
// something to announce.
//
// While it was opt-in, a scan said so once: a Hint describing the deviation the caller had chosen, and
// a Warning when the option was asked for under a loader that cannot honour it. Both were reasonable
// then and neither is now — the first fires on every scan, and nobody asks for a default.
//
// So scan.compiled-dependencies is reserved for the one thing that can still cost the output: a lookup
// that wanted a declaration and found no source to read it from. That is reportSourcelessLookup, and
// TestExportOnly_ReportedWhereItCosts covers it. Reaching the code any other way is noise, which is
// what this pins.
func TestCompiledDependencies_DefaultSaysNothing(t *testing.T) {
	t.Parallel()

	collect := func(t *testing.T, apply func(*codescan.Options)) []codescan.Diagnostic {
		t.Helper()

		var seen []codescan.Diagnostic
		opts := &codescan.Options{
			Packages:   []string{"./petstore/..."},
			WorkDir:    scantest.FixturesDir() + "/goparsing",
			ScanModels: true,
			OnDiagnostic: func(d codescan.Diagnostic) {
				if d.Code == "scan.compiled-dependencies" {
					seen = append(seen, d)
				}
			},
		}
		apply(opts)
		_, err := codescan.Run(opts)
		require.NoError(t, err)

		return seen
	}

	t.Run("the default scan is silent about it", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, collect(t, func(*codescan.Options) {}),
			"every dependency's source is reachable here, so nothing was lost and there is nothing to say")
	})

	t.Run("opting out is silent too", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, collect(t, func(o *codescan.Options) { o.SkipCompiledDependencies = true }))
	})

	// A loader that cannot honour the default is not a loader that has anything to report. It resolves
	// imports itself and already decides per dependency whether to read its source, so it arrives at the
	// same place by its own route.
	t.Run("the toolchain-free loader is silent too", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, collect(t, func(o *codescan.Options) { o.ToolchainFreeLoader = true }))
	})

	// The tree here is deliberately not a working one and must not be read as a model for using FS: an
	// io/fs is the WHOLE world a scan may read, and a module rooted on its own reaches neither GOROOT
	// nor the module cache, so every dependency including the standard library synthesizes.
	//
	// It earns its place because FS forces the toolchain-free loader without the caller naming one, and
	// a scan whose dependencies all synthesized is exactly where a stale announcement would resurface.
	t.Run("a virtual filesystem forces the other loader, still silent", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, collect(t, func(o *codescan.Options) {
			o.FS = os.DirFS(scantest.FixturesDir())
			o.WorkDir = "."
			o.Packages = []string{"./goparsing/petstore/..."}
		}))
	})
}
