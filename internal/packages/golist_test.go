// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package packages_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-openapi/codescan/internal/packages"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// diskTree writes the tagTree fixture to a real directory, since `go list` can only read one.
func diskTree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for name, file := range tagTree() {
		path := filepath.Join(root, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, file.Data, 0o600))
	}

	return root
}

// loadedVia returns the base names of the files a strategy decided ./pkg is made of.
func loadedVia(t *testing.T, root string, opts ...packages.Option) []string {
	t.Helper()

	pkgs, err := packages.NewLoader(opts...).Load(&packages.Config{Dir: root}, "./pkg")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	names := make([]string, 0, len(pkgs[0].GoFiles))
	for _, f := range pkgs[0].GoFiles {
		names = append(names, filepath.Base(f))
	}
	sort.Strings(names)

	return names
}

// TestGoPackagesStrategyHonoursTarget is the asymmetry this file exists to prevent.
//
// The go command takes GOOS/GOARCH from the environment and nowhere else, so a loader that merely
// stores the pinned target would apply it under one strategy and silently ignore it under the other — the
// same Loader answering two different questions depending on how it resolves the graph.
func TestGoPackagesStrategyHonoursTarget(t *testing.T) {
	t.Parallel()

	root := diskTree(t)

	got := loadedVia(t, root, packages.WithGoEnv(packages.GoEnv{GOOS: "windows", GOARCH: "amd64"}))

	assert.Contains(t, got, "impl_windows.go", "the requested target selects the windows file")
	assert.NotContains(t, got, "impl_linux.go", "and drops the one for another platform")
}

// TestGoPackagesStrategyIsTheDefault pins the zero value: a Loader asked for nothing must behave as
// codescan always has.
func TestGoPackagesStrategyIsTheDefault(t *testing.T) {
	t.Parallel()

	root := diskTree(t)

	assert.Equal(t,
		loadedVia(t, root, packages.WithStrategy(packages.StrategyGoPackages), packages.WithGoEnv(packages.GoEnv{GOOS: "windows", GOARCH: "amd64"})),
		loadedVia(t, root, packages.WithGoEnv(packages.GoEnv{GOOS: "windows", GOARCH: "amd64"})))
}

// TestStrategiesAgreeOnTheSameTree runs one tree through both strategies. They resolve the graph in
// completely different ways, so agreeing on which files a package is made of is the whole contract.
func TestStrategiesAgreeOnTheSameTree(t *testing.T) {
	t.Parallel()

	root := diskTree(t)
	target := packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"})

	assert.Equal(t,
		loadedVia(t, root, target),
		loadedVia(t, root, target, packages.WithStrategy(packages.StrategyToolchainFree)),
		"the two strategies must build the same package from the same tree")
}

// TestWithFSOverridesTheStrategy: asking for the go command while handing over a virtual filesystem
// is not a configuration to honour, it is one to correct — `go list` cannot read fsys at all.
func TestWithFSOverridesTheStrategy(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.NewLoader(
		packages.WithStrategy(packages.StrategyGoPackages),
		packages.WithFS(tagTree()),
		packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}),
	).Load(&packages.Config{}, "./pkg")

	require.NoError(t, err, "a virtual filesystem selects the strategy that can read it")
	require.Len(t, pkgs, 1)
	assert.Equal(t, "example.com/tagtree/pkg", pkgs[0].PkgPath)
}
