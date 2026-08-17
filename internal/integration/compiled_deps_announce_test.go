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

// Taking dependency types from compiled export data is something a caller asks for, so a scan says
// whether the request was met: a Hint when the load took the shortcut, a Warning when the resolved
// loader could not.
//
// Only a caller who asked hears any of it. An ordinary scan reads every dependency from source, which
// is not a deviation from anything and not worth a line on every run.
//
// The remaining use of scan.compiled-dependencies is the one thing that can still cost the output: a
// lookup that wanted a declaration and found no source to read it from. That is reportSourcelessLookup,
// covered by TestExportOnly_ReportedWhereItCosts.
func TestCompiledDependencies_AnnouncesWhetherItHappened(t *testing.T) {
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
			"nothing was asked for and every dependency was read as usual, so there is nothing to say")
	})

	t.Run("the default is silent under the toolchain-free loader too", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, collect(t, func(o *codescan.Options) { o.ToolchainFreeLoader = true }),
			"a loader that could not have honoured the option has nothing to report when nobody asked")
	})

	t.Run("asking for it, and getting it", func(t *testing.T) {
		t.Parallel()

		seen := collect(t, func(o *codescan.Options) { o.CompiledDependencies = true })
		require.Len(t, seen, 1)
		assert.Equal(t, codescan.SeverityHint, seen[0].Severity)
		assert.Contains(t, seen[0].Message, "dependency types come from compiled export data")
	})

	// Asked for and not delivered. The toolchain-free loader resolves imports itself and already decides
	// per dependency whether to read its source, so it arrives at the same document by its own route —
	// but the caller chose this for the speed-up and did not get it, and nothing else would say so.
	t.Run("asking for it under a loader that cannot", func(t *testing.T) {
		t.Parallel()

		seen := collect(t, func(o *codescan.Options) {
			o.CompiledDependencies = true
			o.ToolchainFreeLoader = true
		})
		require.Len(t, seen, 1)
		assert.Equal(t, codescan.SeverityWarning, seen[0].Severity)
		assert.Contains(t, seen[0].Message, "is ignored under the")
	})

	// The tree here is deliberately not a working one and must not be read as a model for using FS: an
	// io/fs is the WHOLE world a scan may read, and a module rooted on its own reaches neither GOROOT
	// nor the module cache, so every dependency including the standard library synthesizes.
	//
	// It earns its place because FS forces the toolchain-free loader without the caller naming one, so
	// the warning has to be driven by the loader that RAN rather than by the one that was asked for.
	t.Run("a virtual filesystem forces the other loader, and is warned about", func(t *testing.T) {
		t.Parallel()

		seen := collect(t, func(o *codescan.Options) {
			o.CompiledDependencies = true
			o.FS = os.DirFS(scantest.FixturesDir())
			o.WorkDir = "."
			o.Packages = []string{"./goparsing/petstore/..."}
		})
		require.Len(t, seen, 1)
		assert.Equal(t, codescan.SeverityWarning, seen[0].Severity)
	})
}
