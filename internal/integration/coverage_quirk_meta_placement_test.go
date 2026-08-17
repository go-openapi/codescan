// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// scanQuirkMetaPlacement scans one of the fixture's packages, asking for provenance.
//
// Provenance is what makes the second case below a crash rather than a wrong answer: the info
// node's cross-reference anchor is taken from the meta block's own position, and there was no
// block to take one from. A caller that does not ask for it never reaches that line, which is why
// the TUI hit this and the library did not.
func scanQuirkMetaPlacement(t *testing.T, pkg string) *codescan.Options {
	t.Helper()

	return &codescan.Options{
		Packages:     []string{"./quirks/meta-placement/" + pkg + "/..."},
		WorkDir:      scantest.FixturesDir(),
		OnProvenance: func(scanner.Provenance) {},
		ScanModels:   true,
	}
}

// TestQuirk_MetaPlacement covers a swagger:meta block written somewhere other than the package doc
// comment.
//
// Detection reads every comment in the file, so the annotation was always found; the block was
// then taken from the file's PACKAGE doc regardless of where it had been found. What that produced
// depended only on whether the file happened to have one.
func TestQuirk_MetaPlacement(t *testing.T) {
	// Wrong answer: the authored block is ignored, and an ordinary sentence about the package is
	// parsed as the meta block in its place.
	t.Run("below a package doc", func(t *testing.T) {
		doc, err := codescan.Run(scanQuirkMetaPlacement(t, "below-package-doc"))
		require.NoError(t, err, "a meta block below the package clause must scan")
		require.NotNil(t, doc)
		require.NotNil(t, doc.Info)

		assert.Equal(t, "1.2.3", doc.Info.Version)
		assert.Equal(t, "api.example.com", doc.Host)
		assert.Equal(t, "/v2", doc.BasePath)
		assert.Equal(t, []string{"https"}, doc.Schemes)

		// The package doc is prose about the package, not about the API, so none of it may show up
		// as the specification's own title or description.
		assert.NotContains(t, doc.Info.Title, "package doc")
		assert.NotContains(t, doc.Info.Description, "package doc")

		scantest.CompareOrDumpJSON(t, doc, "quirk_meta_placement.json")
	})

	// No answer at all: the file's doc is nil, and the anchor recorded for the info node
	// dereferenced it.
	t.Run("with no package doc", func(t *testing.T) {
		doc, err := codescan.Run(scanQuirkMetaPlacement(t, "no-package-doc"))
		require.NoError(t, err, "a file with no package doc must scan")
		require.NotNil(t, doc)
		require.NotNil(t, doc.Info)

		assert.Equal(t, "4.5.6", doc.Info.Version)
		assert.Equal(t, "nodoc.example.com", doc.Host)

		scantest.CompareOrDumpJSON(t, doc, "quirk_meta_placement_no_package_doc.json")
	})
}
