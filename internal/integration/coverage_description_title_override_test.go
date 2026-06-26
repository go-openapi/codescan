// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_DescriptionTitleOverride_Schema exercises P2 of the swagger:title /
// swagger:description override feature (Q30 close-out): on a model and on struct fields, an
// explicit override replaces the godoc-derived title / description; absence leaves the godoc
// untouched; a bare swagger:description suppresses the godoc (empty applied) and raises
// scan.empty-override.
func TestCoverage_DescriptionTitleOverride_Schema(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/description-title-override/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	require.MapContainsT(t, doc.Definitions, "Widget")
	widget := doc.Definitions["Widget"]

	// Model: both title and description overridden (godoc text gone).
	assert.EqualT(t, "A Public Widget", widget.Title)
	assert.EqualT(t, "A widget exposed via the public API.", widget.Description)
	assert.NotEqualT(t, "Widget is the Go-facing widget doc, written for Go readers.", widget.Title)

	require.MapContainsT(t, widget.Properties, "id")
	require.MapContainsT(t, widget.Properties, "label")
	require.MapContainsT(t, widget.Properties, "plain")
	require.MapContainsT(t, widget.Properties, "suppressed")

	// Field: description override; fields carry no title by default.
	id := widget.Properties["id"]
	assert.EqualT(t, "The unique widget identifier.", id.Description)
	assert.EqualT(t, "", id.Title)

	// Field: title override (the only way a property gets a title) + description.
	label := widget.Properties["label"]
	assert.EqualT(t, "Display Label", label.Title)
	assert.EqualT(t, "Human-readable label shown to API consumers.", label.Description)

	// Regression: no override → godoc description retained.
	plain := widget.Properties["plain"]
	assert.NotEqualT(t, "", plain.Description)

	// Override + inline validation keyword coexist on the same field: the description override applies
	// AND maximum survives (schema-family dispatch).
	capacity := widget.Properties["capacity"]
	assert.EqualT(t, "The maximum capacity, in liters.", capacity.Description)
	require.NotNil(t, capacity.Maximum)
	assert.EqualT(t, float64(1000), *capacity.Maximum)

	// Multi-line description override (Option B): the body lines fold into one description, joined
	// with newlines, terminated by the blank line.
	notes := widget.Properties["notes"]
	assert.EqualT(t,
		"Free-form notes about the widget.\nThey may span several lines, all folded into one description.",
		notes.Description)

	// Empty override: bare swagger:description suppresses the godoc.
	suppressed := widget.Properties["suppressed"]
	assert.EqualT(t, "", suppressed.Description)

	// $ref field: under the default flags (no EmitRefSiblings, no DescWithRef, no validation forcing a
	// compound), title/description ride description's fate and are dropped to a bare {$ref} — same
	// rule as a prose description.
	require.MapContainsT(t, widget.Properties, "gadget")
	gadget := widget.Properties["gadget"]
	assert.NotEqualT(t, "", gadget.Ref.String())
	assert.EqualT(t, "", gadget.Title)
	assert.EqualT(t, "", gadget.Description)

	var sawEmptyOverride bool
	for _, d := range diags {
		if d.Code == grammar.CodeEmptyOverride {
			sawEmptyOverride = true
		}
	}
	assert.TrueT(t, sawEmptyOverride, "expected scan.empty-override for the bare swagger:description")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_description_title_override.json")
}

// TestCoverage_DescriptionTitleOverride_RefSiblings witnesses the symmetric $ref-sibling threading:
// with EmitRefSiblings the $ref field's title and description overrides are preserved as direct
// siblings of the $ref ({$ref, title, description}) rather than dropped — the override flows
// through the same machinery a prose description would.
func TestCoverage_DescriptionTitleOverride_RefSiblings(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:        []string{"./enhancements/description-title-override/..."},
		WorkDir:         scantest.FixturesDir(),
		ScanModels:      true,
		EmitRefSiblings: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	require.MapContainsT(t, doc.Definitions, "Widget")
	gadget := doc.Definitions["Widget"].Properties["gadget"]
	assert.NotEqualT(t, "", gadget.Ref.String(), "the $ref is retained")
	assert.EqualT(t, "Gadget Ref", gadget.Title, "title override preserved as a $ref sibling")
	assert.EqualT(t, "The attached gadget, described for API consumers.", gadget.Description,
		"description override preserved as a $ref sibling")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_description_title_override_ref_siblings.json")
}

// TestCoverage_DescriptionTitleOverride_Responses exercises P3: a swagger:description override on a
// response declaration and on a response header replaces the godoc; a swagger:title on a response
// is rejected with scan.context-invalid (OpenAPI 2.0 has no Response/Header title) while the
// description override still applies.
func TestCoverage_DescriptionTitleOverride_Responses(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/description-title-override-responses/..."},
		WorkDir:  scantest.FixturesDir(),
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	require.MapContainsT(t, doc.Responses, "errorResponse")
	resp := doc.Responses["errorResponse"]

	// Response description overridden (godoc gone).
	assert.EqualT(t, "The error payload returned to API consumers.", resp.Description)

	// Header description overridden.
	require.MapContainsT(t, resp.Headers, "X-Error-Code")
	assert.EqualT(t, "The machine-readable error code.", resp.Headers["X-Error-Code"].Description)

	// swagger:title on the response is rejected as context-invalid.
	var sawContextInvalid bool
	for _, d := range diags {
		if d.Code == grammar.CodeContextInvalid {
			sawContextInvalid = true
		}
	}
	assert.TrueT(t, sawContextInvalid, "expected scan.context-invalid for swagger:title on a response")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_description_title_override_responses.json")
}
