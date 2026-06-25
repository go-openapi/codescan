// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func runGodocLinks(t *testing.T, on bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/godoc-links/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		CleanGoDoc: on,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestCoverage_GodocLinks_Off is the control: with CleanGoDoc off, godoc
// doc-link syntax is carried into the spec verbatim (byte-identical to the
// historic behaviour). The golden pins the full untouched output.
func TestCoverage_GodocLinks_Off(t *testing.T) {
	doc := runGodocLinks(t, false)

	widget := doc.Definitions["Widget"]
	// Brackets are preserved when the option is off.
	assert.Contains(t, widget.Title+widget.Description, "[Person]")
	assert.Contains(t, widget.Description, "[the spec]: https://example.com/spec")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_godoc_links_off.json")
}

// TestCoverage_GodocLinks_On exercises P1: doc-link brackets are stripped and
// the leaf identifier humanized; reference-definition lines are dropped; the
// conservative recognizer leaves non-doc-link brackets intact. (Resolution to
// the exposed schema name is a later phase — here every link is humanized.)
func TestCoverage_GodocLinks_On(t *testing.T) {
	doc := runGodocLinks(t, true)

	widget := doc.Definitions["Widget"]
	prose := widget.Title + " " + widget.Description

	// doc-links stripped + humanized; no brackets survive on real links.
	assert.Contains(t, prose, "person")
	assert.Contains(t, prose, "ledger")
	assert.NotContains(t, prose, "[Person]")
	assert.NotContains(t, prose, "[inventory.Ledger]")
	assert.NotContains(t, prose, "[*Widget]")

	// reference-definition line dropped entirely.
	assert.NotContains(t, widget.Description, "the spec")
	assert.NotContains(t, widget.Description, "https://example.com/spec")

	// field doc-link humanized.
	assert.Contains(t, widget.Properties["holder"].Description, "person")
	assert.NotContains(t, widget.Properties["holder"].Description, "[Person]")

	// conservative recognizer: non-doc-link brackets are left intact.
	idx := widget.Properties["index"].Description
	assert.Contains(t, idx, "[0]")
	assert.Contains(t, idx, "[see notes]")
	assert.Contains(t, idx, "[id]")

	// dotted type.field leaf humanized.
	assert.Contains(t, doc.Definitions["Person"].Properties["custName"].Description, "cust name")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_godoc_links.json")
}
