// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scantest

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/require"
)

var (
	petstoreCtx            *scanner.ScanCtx //nolint:gochecknoglobals // test package cache shared across test functions
	classificationCtx      *scanner.ScanCtx //nolint:gochecknoglobals // test package cache shared across test functions
	go118ClassificationCtx *scanner.ScanCtx //nolint:gochecknoglobals // test package cache shared across test functions
)

// FixturesDir returns the absolute path to the repo-level fixtures/ directory,
// so tests can run from any package depth without fragile relative paths.
func FixturesDir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("scantest: unable to resolve caller for fixtures path")
	}
	// thisFile is <repo>/internal/scantest/load.go
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "fixtures"))
}

func LoadPetstorePkgsCtx(t testing.TB, enableDebug bool) *scanner.ScanCtx {
	t.Helper()

	if petstoreCtx != nil {
		return petstoreCtx
	}

	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages: []string{"./goparsing/petstore/..."},
		WorkDir:  FixturesDir(),
		Debug:    enableDebug,
	})

	require.NoError(t, err)
	petstoreCtx = ctx

	return petstoreCtx
}

func LoadGo118ClassificationPkgsCtx(t *testing.T) *scanner.ScanCtx {
	t.Helper()

	if go118ClassificationCtx != nil {
		return go118ClassificationCtx
	}

	sctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages: []string{
			"./goparsing/go118",
		},
		WorkDir: FixturesDir(),
	})
	require.NoError(t, err)
	go118ClassificationCtx = sctx

	return go118ClassificationCtx
}

func LoadClassificationPkgsCtx(t *testing.T) *scanner.ScanCtx {
	t.Helper()

	if classificationCtx != nil {
		return classificationCtx
	}

	sctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages: []string{
			"./goparsing/classification",
			"./goparsing/classification/models",
			"./goparsing/classification/operations",
		},
		WorkDir: FixturesDir(),
	})
	require.NoError(t, err)
	classificationCtx = sctx

	return classificationCtx
}
