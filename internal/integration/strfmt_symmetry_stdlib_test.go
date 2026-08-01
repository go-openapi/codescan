// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import "testing"

// F4 of the strfmt dispatch-symmetry matrix: precedence between a user's format annotation and the
// builder's own stdlib recognizers.
//
// The contract is that the AUTHOR ALWAYS WINS. `swagger:strfmt` is the escape hatch for exactly the
// case the library cannot infer — a time.Time may go on the wire as `date`, or as some custom
// format nothing can guess — so a recognizer is a default for un-annotated code, never an override
// of an explicit annotation.
//
// Today only the named half honours that. It never reaches a recognizer at all
// (applyStdlibSpecials is keyed on the declaration's own identity, and `StampNamed` is not
// `time.Time`), so its classifier wins by construction rather than by precedence. The alias half
// dissolves onto the stdlib type itself, the recognizer fires first, and the annotation loses to a
// confidently wrong answer: `date-time` for the time pair, and for json.RawMessage an untyped `{}`
// — the open "any JSON" shape, right as a default and precisely wrong as an override.
//
// See strfmt_symmetry_harness_test.go for how the ledger reads.
func TestStrfmtSymmetryStdlib(t *testing.T) {
	ledger := symmetryLedger{
		pkg:          "enhancements/strfmt-symmetry-stdlib",
		goldenPrefix: "strfmt_symmetry_stdlib",
		cells: []symmetryCell{
			{
				definition: "Envelope", namedProp: "fieldTimeNamed", aliasProp: "fieldTimeAlias",
				wantNamed: "string/date",
				note:      "alias yields the recognizer's date-time, NOT the author's date",
			},
			{
				definition: "Envelope", namedProp: "fieldRawNamed", aliasProp: "fieldRawAlias",
				wantNamed: "string/byte",
				note:      "alias yields recognizeRawMessage's untyped open schema, dropping the format AND the type",
			},
		},

		exceptions:    map[string]string{},
		knownBroken:   map[string]string{},
		controlBroken: map[string]string{},
	}

	ledger.run(t)
}
