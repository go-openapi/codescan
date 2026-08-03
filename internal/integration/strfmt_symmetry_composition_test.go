// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import "testing"

// F2 of the strfmt dispatch-symmetry matrix: the two composition dispatch sites that bypass
// `buildFromType` — plain struct embed (`buildEmbedded`) and allOf member (`buildAllOf`).
//
// Cells are compared at DEFINITION level: a composition has no enclosing property to inspect.
//
// See strfmt_symmetry_harness_test.go for how the ledger reads.
func TestStrfmtSymmetryComposition(t *testing.T) {
	ledger := symmetryLedger{
		pkg:          "enhancements/strfmt-symmetry-composition",
		goldenPrefix: "strfmt_symmetry_composition",
		cells: []symmetryCell{
			// Plain embed of a BASIC-underlying type: the member no longer vanishes. Such an embed
			// promotes nothing, so it is an ordinary property keyed by the Go field name and built from
			// the embedded type — the format included. The two halves therefore differ by the only thing
			// that legitimately differs between them, the identifier being embedded, which is why this is
			// an exception rather than a symmetry failure. That the format lands is asserted where the
			// signature can show it, in TestEmbedBasicUnderlying.
			{
				namedProp: "EmbedBasicNamed", aliasProp: "EmbedBasicAlias",
				wantNamed: "object{FmtBasicNamed,label}",
			},
			// Plain embed of a STRUCT-underlying type: still symmetric, still a shared gap. This one
			// really does promote, and no arm of the promotion walk consults the embedded type's format —
			// left unasserted because what a formatted type SHOULD contribute when its properties are
			// promoted is an open question, not a bug with one answer.
			{
				namedProp: "EmbedStructNamed", aliasProp: "EmbedStructAlias",
				note: "SHARED GAP: both halves promote left/right and drop the format",
			},

			// allOf member: the money row. The named arm runs classifierAliasTargetStrfmt (allof.go:205);
			// the alias arm drops straight into buildAlias and dissolves.
			{
				namedProp: "AllOfBasicNamed", aliasProp: "AllOfBasicAlias",
				wantNamed: "allOf[string/isbn+object{note}]",
			},
			{
				namedProp: "AllOfStructNamed", aliasProp: "AllOfStructAlias",
				wantNamed: "allOf[string/duration+object{note}]",
			},
		},

		// A promotes-nothing embed is keyed by the embedded IDENTIFIER, and the two halves of a pair
		// are two different identifiers by construction. The difference is the fixture's, not the
		// builder's.
		exceptions: map[string]string{
			"default/EmbedBasic":            "the property is named after the embedded identifier",
			"refaliases/EmbedBasic":         "the property is named after the embedded identifier",
			"transparentaliases/EmbedBasic": "the property is named after the embedded identifier",
		},
		// The allOf alias arm now reads the member's declaration before dissolving, matching the
		// classifierAliasTargetStrfmt its named counterpart runs.
		knownBroken:   map[string]string{},
		controlBroken: map[string]string{},
	}

	ledger.run(t)
}
