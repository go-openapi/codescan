// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner_test

import (
	"encoding/json"
	"testing"

	"github.com/go-openapi/codescan/internal/builders/spec"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestSpecIsBuiltWithoutExpressionTypes is the witness that the builders no longer need
// types.Info.Types.
//
// That map records what each type EXPRESSION denotes, and only a source type-check produces it: the
// field distinguishing a type from a value is unexported, so it cannot be rebuilt by hand for a
// package whose types came from compiled export data. While the builders read it, such a package
// cannot be handed to them with its source attached — which is the constraint the loader currently
// works around by choosing, per dependency, between export data and a full type-check.
//
// So the whole spec is built twice over each corpus, once with the map dropped, and the two
// documents compared. Dropping it used to panic on a plain swagger:model and to lose a
// discriminated subtype from the reverse allOf index; both are covered below.
func TestSpecIsBuiltWithoutExpressionTypes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		packages []string
		// contains names one definition the corpus must emit, so a comparison of two empty
		// documents cannot pass for agreement.
		contains string
	}{
		{
			// The reverse allOf index: a subtype $refs its base and nothing $refs the subtype, so an
			// embed the index cannot resolve drops a whole polymorphic family in silence.
			name:     "a discriminated family",
			packages: []string{"./enhancements/discriminated-subtypes/..."},
			contains: "modelX",
		},
		{
			name:     "a multi-level discriminated hierarchy",
			packages: []string{"./enhancements/discriminated-subtypes-nested/..."},
			contains: "Square",
		},
		{
			// Named types written over other named types (`type SomeTimeType time.Time`), interface
			// embeds, aliases: everything that reads a declaration's written right-hand side.
			name:     "the classification corpus",
			packages: []string{"./goparsing/classification/..."},
			contains: "User",
		},
		{
			name:     "the petstore",
			packages: []string{"./goparsing/petstore/..."},
			contains: "Pet",
		},
		{
			name:     "interface methods and embeds",
			packages: []string{"./enhancements/interface-methods/..."},
		},
		{
			name:     "alias declarations",
			packages: []string{"./enhancements/alias-expand/..."},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := buildCorpus(t, tc.packages, false)
			got := buildCorpus(t, tc.packages, true)

			if tc.contains != "" {
				assert.Contains(t, want, tc.contains,
					"the corpus does not emit %q, so the comparison would not bite", tc.contains)
			}
			assert.EqualT(t, want, got)
		})
	}
}

// buildCorpus scans packages and returns the emitted document as JSON, optionally after dropping
// every package's record of what its type expressions denote.
func buildCorpus(t *testing.T, pkgs []string, dropExpressionTypes bool) string {
	t.Helper()

	ctx, err := scanner.NewScanCtx(applyLoader(&scanner.Options{
		Packages:   pkgs,
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	}))
	require.NoError(t, err)

	if dropExpressionTypes {
		ctx.DropExpressionTypes()
	}

	doc, err := spec.NewBuilder(nil, ctx, true).Build()
	require.NoError(t, err)

	out, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)

	return string(out)
}

func applyLoader(opts *scanner.Options) *scanner.Options {
	return scantest.ApplyLoader(opts)
}
