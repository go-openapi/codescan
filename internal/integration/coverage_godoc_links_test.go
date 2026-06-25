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

	gizmo := doc.Definitions["gizmo"] // swagger:model gizmo on type Widget
	// brackets and the reference-definition line are preserved when off.
	assert.Contains(t, gizmo.Title, "[Gadget]")
	assert.Contains(t, gizmo.Title, "[Order.CustName]")
	assert.Contains(t, gizmo.Description, "[the spec]: https://example.com/spec")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_godoc_links_off.json")
}

// TestCoverage_GodocLinks_On exercises the full feature: resolvable doc-links
// are recomposed to the referenced schema's EXPOSED name, unresolved links are
// humanized, reference-definition lines are dropped, and non-doc-link brackets
// are left intact.
func TestCoverage_GodocLinks_On(t *testing.T) {
	doc := runGodocLinks(t, true)

	gizmo := doc.Definitions["gizmo"]

	// leading self-name recomposed to the exposed model name (swagger:model
	// gizmo), restored to sentence case: "Widget ..." -> "Gizmo ...".
	assert.Contains(t, gizmo.Title, "Gizmo is the primary")
	assert.NotContains(t, gizmo.Title, "Widget")
	// doc-link to another model -> its exposed name (no override -> Go name).
	assert.Contains(t, gizmo.Title, "Gadget")
	assert.NotContains(t, gizmo.Title, "[Gadget]")
	// type.field doc-link -> exposed model name + exposed (json) property name.
	assert.Contains(t, gizmo.Title, "Order.customer_name")
	assert.NotContains(t, gizmo.Title, "[Order.CustName]")

	// reference-definition line dropped; pointer link resolved; unknown link
	// humanized (not a scanned model).
	assert.NotContains(t, gizmo.Description, "the spec")
	assert.NotContains(t, gizmo.Description, "https://example.com/spec")
	assert.Contains(t, gizmo.Description, "Gadget pointer")
	assert.Contains(t, gizmo.Description, "unknown sprocket")
	assert.NotContains(t, gizmo.Description, "[")
	// cross-package doc-link ([inventory.Ledger]) resolved to its exposed name
	// via the file's imports.
	assert.Contains(t, gizmo.Description, "Ledger")
	require.MapContainsT(t, doc.Definitions, "Ledger")

	// field doc-link recomposed; leading field self-name is left as-is (only a
	// declaration's own leading name is recomposed).
	assert.Contains(t, gizmo.Properties["holder"].Description, "Gadget that owns")
	assert.NotContains(t, gizmo.Properties["holder"].Description, "[Gadget]")

	// conservative recognizer: non-doc-link brackets are left intact.
	idx := gizmo.Properties["index"].Description
	assert.Contains(t, idx, "[0]")
	assert.Contains(t, idx, "[see notes]")
	assert.Contains(t, idx, "[id]")

	// no markers ever leak into the output.
	assert.NotContains(t, gizmo.Title+gizmo.Description, "\x00")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_godoc_links.json")
}
