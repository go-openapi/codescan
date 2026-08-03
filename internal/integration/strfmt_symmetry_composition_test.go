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
			// Plain embed: SYMMETRIC, and both halves are wrong the same way. buildNamedEmbedded switches
			// on the member's underlying shape and never consults its comments, so the format is dropped on
			// both sides — a basic member vanishes entirely, a struct member promotes its properties. Not a
			// Q32 asymmetry; the same shared gap Q33 describes for TextMarshaler embeds. Left unasserted
			// because what an embed of a formatted type SHOULD produce is an open design question.
			{
				namedProp: "EmbedBasicNamed", aliasProp: "EmbedBasicAlias",
				note: "SHARED GAP: both halves drop the member entirely (buildNamedEmbedded reads no comments) — see Q33",
			},
			{
				namedProp: "EmbedStructNamed", aliasProp: "EmbedStructAlias",
				note: "SHARED GAP: both halves promote left/right and drop the format — see Q33",
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

		exceptions: map[string]string{},
		// The allOf alias arm now reads the member's declaration before dissolving, matching the
		// classifierAliasTargetStrfmt its named counterpart runs.
		knownBroken:   map[string]string{},
		controlBroken: map[string]string{},
	}

	ledger.run(t)
}
