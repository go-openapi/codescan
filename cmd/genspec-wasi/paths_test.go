// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestAbsolutePathLeavesAnAbsolutePathAlone is the Windows regression: -workdir is handed to the
// toolchain-free loader as the place patterns resolve against, and a path recognised as relative is
// appended to the working directory. "D:\tmp\x" does not start with a slash, so the leading-slash
// rule made every absolute Windows -workdir name a directory that cannot exist.
func TestAbsolutePathLeavesAnAbsolutePathAlone(t *testing.T) {
	t.Parallel()

	dir := t.TempDir() // absolute in whatever shape this host writes

	got, err := absolutePath(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, got)
}

// TestAbsolutePathAnchorsARelativePath covers the other half: what is relative is resolved, and the
// answer is always absolute, since that is the whole contract the loader relies on.
func TestAbsolutePathAnchorsARelativePath(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	require.NoError(t, err)

	for _, p := range []string{".", "", "./models", "models", "a/../b"} {
		got, err := absolutePath(p)
		require.NoError(t, err)

		assert.True(t, filepath.IsAbs(got), "%q resolved to %q, which is not absolute", p, got)
		assert.Equal(t, filepath.Join(wd, p), got, "%q", p)
		assert.Equal(t, filepath.Clean(got), got, "%q resolved to an uncleaned path", p)
	}
}
