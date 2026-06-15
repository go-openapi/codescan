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

// TestQuirk_ModelOverrideMatrix is the F1/F2/F4 WITNESS (documented-current,
// post-F3): how swagger:model combined with a per-type override
// (swagger:strfmt / swagger:type / swagger:enum) renders today. No fix — the
// golden quirk_model_override_matrix.json is the baseline the reconciliation
// is measured against. The TODO markers flag what should change: each override
// type should publish a first-class definition carrying the FULL override
// schema, and referencing fields should $ref it (instead of inlining + an
// orphan, degenerate definition).
func TestQuirk_ModelOverrideMatrix(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:     []string{"./quirks/model-override-matrix/..."},
		WorkDir:      scantest.FixturesDir(),
		ScanModels:   true,
		OnDiagnostic: func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	h := doc.Definitions["Holder"].Properties
	require.NotNil(t, h)

	// TODO F1: strfmt+model inlines {string,uuid} on the field; the UUID
	// definition is an orphan {type:string} with the format DROPPED. Should
	// be a {string,format:uuid} definition the field $refs.
	id := h["id"]
	assert.Equal(t, []string{"string"}, []string(id.Type))
	assert.Equal(t, "uuid", id.Format)
	assert.Empty(t, id.Ref.String(), "TODO F1: field should $ref UUID")
	assert.Empty(t, doc.Definitions["UUID"].Format, "TODO F1: definition drops the format today")

	// TODO F2: type+model inlines {string} on the field; RawID definition is
	// an orphan {type:string}. Should be a definition the field $refs.
	raw := h["raw"]
	assert.Equal(t, []string{"string"}, []string(raw.Type))
	assert.Empty(t, raw.Ref.String(), "TODO F2: field should $ref RawID")

	// TODO F4: enum+model inlines the values on the field; the Status
	// definition is an orphan {type:string} with NO enum. Should be an enum
	// definition the field $refs.
	state := h["state"]
	assert.Equal(t, []any{"active", "closed"}, state.Enum)
	assert.Empty(t, state.Ref.String(), "TODO F4: field should $ref Status")
	assert.Empty(t, doc.Definitions["Status"].Enum, "TODO F4: definition has no enum today")

	// F4b: BARE swagger:enum (no name) is a parse error — the enum is dropped
	// entirely, so Priority is a plain integer model the field $refs.
	priority := h["priority"]
	assert.Equal(t, "#/definitions/Priority", priority.Ref.String())
	assert.Empty(t, doc.Definitions["Priority"].Enum, "TODO F4: bare swagger:enum collects no values today")

	var bareEnumErr bool
	for _, d := range diags {
		if d.Code == grammar.CodeMissingRequiredArg {
			bareEnumErr = true
		}
	}
	assert.True(t, bareEnumErr, "bare swagger:enum currently errors (name/value-list required)")

	scantest.CompareOrDumpJSON(t, doc, "quirk_model_override_matrix.json")
}
