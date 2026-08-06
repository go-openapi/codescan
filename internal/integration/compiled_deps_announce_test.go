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

// Only one strategy can take dependency types from the compiler, and the caller does not always get
// the strategy they asked for — a virtual filesystem forces the toolchain-free one whatever else was
// requested. So what the scan announces has to come from the strategy that ran, not from the field
// that was set.
//
// This used to announce the request. A scan combining CompiledDependencies with the toolchain-free
// loader was told its dependency types came from export data while it read every dependency from
// source: a diagnostic contradicting the load it described, on the one channel this scanner asks
// callers to trust.
func TestCompiledDependencies_AnnouncedFromWhatRan(t *testing.T) {
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

	t.Run("honoured: a hint describing what the load did", func(t *testing.T) {
		t.Parallel()

		seen := collect(t, func(o *codescan.Options) { o.CompiledDependencies = true })

		require.NotEmpty(t, seen)
		assert.Equal(t, codescan.SeverityHint, seen[0].Severity)
		assert.Contains(t, seen[0].Message, "come from compiled export data")
	})

	t.Run("ignored: a warning saying so", func(t *testing.T) {
		t.Parallel()

		seen := collect(t, func(o *codescan.Options) {
			o.ToolchainFreeLoader = true
			o.CompiledDependencies = true
		})

		require.Len(t, seen, 1, "the option had no effect, and that is the whole of what there is to say")
		assert.Equal(t, codescan.SeverityWarning, seen[0].Severity,
			"a request that did nothing is more than informational")
		assert.Contains(t, seen[0].Message, "is ignored under")
		assert.NotContains(t, seen[0].Message, "come from compiled export data",
			"the load read every dependency from source; saying otherwise is what this test exists for")
	})

	// The override that needs no loader mentioned at all.
	//
	// The tree here is deliberately not a working one and must not be read as a model for using FS: an
	// io/fs is the WHOLE world a scan may read, and a module rooted on its own reaches neither GOROOT
	// nor the module cache — absolute paths are resolved inside the FS, so every dependency including
	// the standard library synthesizes. That is beside the point being pinned, which is that FS decides
	// the strategy and the announcement follows the strategy.
	t.Run("ignored because a virtual filesystem forced the loader", func(t *testing.T) {
		t.Parallel()

		seen := collect(t, func(o *codescan.Options) {
			o.FS = os.DirFS(scantest.FixturesDir())
			o.WorkDir = "."
			o.Packages = []string{"./goparsing/petstore/..."}
			o.CompiledDependencies = true
		})

		require.Len(t, seen, 1, "FS overrides the strategy without the caller mentioning a loader at all")
		assert.Equal(t, codescan.SeverityWarning, seen[0].Severity)
	})

	t.Run("not asked for, not mentioned", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, collect(t, func(*codescan.Options) {}))
	})
}
