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

// TestQuirk_ModelOverrideMatrix verifies the F1/F2/F4 reconciliation: a
// swagger:model type carrying a per-type override (swagger:strfmt / swagger:type
// / swagger:enum) now publishes a first-class definition carrying the FULL
// override schema, and referencing fields $ref it (instead of inlining + an
// orphan, degenerate definition). The golden quirk_model_override_matrix.json
// captures the fixed shape.
//
// F4b (bare swagger:enum const collection) is NOT yet done — its assertions
// carry TODO Part C and still pin the current (no-enum) behaviour.
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

	// F1: strfmt+model — field $refs UUID; the UUID definition carries the
	// format ({type:string, format:uuid}), no longer an orphan.
	id := h["id"]
	assert.Equal(t, "#/definitions/UUID", id.Ref.String())
	assert.Empty(t, id.Type, "field is a $ref, not an inlined scalar")
	assert.Equal(t, "uuid", doc.Definitions["UUID"].Format)

	// F2: type+model — field $refs RawID; the RawID definition is {type:string}
	// (the swagger:type override applied to the decl).
	raw := h["raw"]
	assert.Equal(t, "#/definitions/RawID", raw.Ref.String())
	assert.Equal(t, []string{"string"}, []string(doc.Definitions["RawID"].Type))

	// F4: enum+model — field $refs Status; the Status definition carries the
	// enum values, no longer value-less.
	state := h["state"]
	assert.Equal(t, "#/definitions/Status", state.Ref.String())
	assert.Equal(t, []any{"active", "closed"}, doc.Definitions["Status"].Enum)

	// F4b (TODO Part C): bare swagger:enum (no name) still does not collect the
	// decl's consts — Priority is a plain integer model with no enum, and the
	// bare form still raises a parse error. Part C will infer the name from the
	// decl and collect the consts.
	priority := h["priority"]
	assert.Equal(t, "#/definitions/Priority", priority.Ref.String())
	assert.Empty(t, doc.Definitions["Priority"].Enum, "TODO Part C: bare swagger:enum collects no values yet")

	var bareEnumErr bool
	for _, d := range diags {
		if d.Code == grammar.CodeMissingRequiredArg {
			bareEnumErr = true
		}
	}
	assert.True(t, bareEnumErr, "TODO Part C: bare swagger:enum still errors (name/value-list required)")

	scantest.CompareOrDumpJSON(t, doc, "quirk_model_override_matrix.json")
}
