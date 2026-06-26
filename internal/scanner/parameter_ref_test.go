// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"slices"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// parametersBlock parses ref.Comments through the grammar and returns the first ParametersBlock —
// the targeting parse lives in the grammar, so the test asserts discovery via the same path a
// builder would consume.
func parametersBlock(t *testing.T, sctx *ScanCtx, ref *ParameterRef) *grammar.ParametersBlock {
	t.Helper()
	for _, b := range grammar.NewParser(sctx.FileSet()).ParseAll(ref.Comments) {
		if pb, ok := b.(*grammar.ParametersBlock); ok {
			return pb
		}
	}
	t.Fatalf("no ParametersBlock parsed from discovered reference marker")
	return nil
}

// TestParameterRefDiscovery verifies the scanner discovers standalone `swagger:parameters`
// reference markers hosted on func declarations — the references that wire shared parameters into
// operations / path-items.
//
// Struct-hosted markers (definitions) must NOT appear among the refs.
func TestParameterRefDiscovery(t *testing.T) {
	t.Run("operation reference on a route func", func(t *testing.T) {
		sctx, err := NewScanCtx(&Options{
			Packages: []string{"./enhancements/shared-parameters"},
			WorkDir:  "../../fixtures",
		})
		require.NoError(t, err)

		refs := slices.Collect(sctx.ParameterRefs())
		// Fixture 1 carries exactly one reference marker:
		//   swagger:parameters listPets X-Request-ID   (on ListPets)
		// every other swagger:parameters is on a struct (a definition).
		require.Len(t, refs, 1)

		pb := parametersBlock(t, sctx, refs[0])
		assert.Equal(t, grammar.ParamTargetOperations, pb.Target)
		assert.Equal(t, []string{"listPets", "X-Request-ID"}, pb.Args)
	})

	t.Run("path reference on a standalone func", func(t *testing.T) {
		sctx, err := NewScanCtx(&Options{
			Packages: []string{"./enhancements/shared-parameters-pathitem"},
			WorkDir:  "../../fixtures",
		})
		require.NoError(t, err)

		refs := slices.Collect(sctx.ParameterRefs())
		// Fixture 2 carries exactly one reference marker:
		//   swagger:parameters /pets X-Request-ID   (on pathItemRefs)
		require.Len(t, refs, 1)

		pb := parametersBlock(t, sctx, refs[0])
		assert.Equal(t, grammar.ParamTargetPath, pb.Target)
		assert.Equal(t, "/pets", pb.Path)
		assert.Equal(t, []string{"X-Request-ID"}, pb.Args)
	})

	t.Run("no reference markers when all swagger:parameters are struct definitions", func(t *testing.T) {
		sctx, err := NewScanCtx(&Options{
			Packages: []string{"./goparsing/classification/operations"},
			WorkDir:  "../../fixtures",
		})
		require.NoError(t, err)

		assert.Empty(t, slices.Collect(sctx.ParameterRefs()))
	})
}

// TestSharedParametersFixturesScanClean is a smoke test that every shared-parameters fixture
// package classifies without a scan error (e.g. no spurious struct-annotation conflict from prose
// in a doc comment).
//
// It guards the fixtures before the later build phases consume them.
func TestSharedParametersFixturesScanClean(t *testing.T) {
	pkgs := []string{
		"./enhancements/shared-parameters",
		"./enhancements/shared-parameters-pathitem",
		"./enhancements/shared-parameters-conflict/pkga",
		"./enhancements/shared-parameters-conflict/pkgb",
		"./enhancements/shared-parameters-yaml",
		"./enhancements/shared-parameters-overrides",
		"./enhancements/shared-parameters-prune",
	}
	for _, pkg := range pkgs {
		t.Run(pkg, func(t *testing.T) {
			_, err := NewScanCtx(&Options{Packages: []string{pkg}, WorkDir: "../../fixtures"})
			assert.NoError(t, err)
		})
	}
}
