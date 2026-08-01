// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import "testing"

// F1 of the strfmt dispatch-symmetry matrix: every cell reachable through `buildFromType` — the
// busiest of `buildAlias`'s four callers (`schema.go:351`).
//
// See strfmt_symmetry_harness_test.go for how the ledger reads.
func TestStrfmtSymmetryCore(t *testing.T) {
	// The use-site dispatch never reads the alias declaration's comments: buildAlias fetches the decl
	// only to test HasModelAnnotation(), then dissolves to Rhs().
	const dissolve = "buildAlias dissolves to Rhs() without consulting the alias decl's comments (schema.go:399-430)"

	// Under TransparentAliases the early return at schema.go:411 precedes the decl lookup entirely, so
	// not even a model annotation can compensate.
	const transparent = "TransparentAliases returns at schema.go:411 before the decl lookup; the model annotation cannot compensate"

	broken := forEveryMode(dissolve,
		"fieldBasic", "fieldStruct", "fieldSlice", "fieldArray",
		"pointerBasic", "pointerStruct",
		"sliceElemBasic", "sliceElemStruct",
		"mapValueBasic", "mapValueStruct",
	)
	broken["transparentaliases/modeledBasic"] = transparent
	broken["transparentaliases/modeledStruct"] = transparent

	ledger := symmetryLedger{
		pkg:          "enhancements/strfmt-symmetry-core",
		goldenPrefix: "strfmt_symmetry_core",
		cells: []symmetryCell{
			{definition: "Envelope", namedProp: "fieldBasicNamed", aliasProp: "fieldBasicAlias", wantNamed: "string/isbn"},
			{definition: "Envelope", namedProp: "fieldStructNamed", aliasProp: "fieldStructAlias", wantNamed: "string/duration"},
			{definition: "Envelope", namedProp: "fieldSliceNamed", aliasProp: "fieldSliceAlias", wantNamed: "string/byte"},
			{definition: "Envelope", namedProp: "fieldArrayNamed", aliasProp: "fieldArrayAlias", wantNamed: "string/bsonobjectid"},
			{
				definition: "Envelope", namedProp: "fieldChainNamed", aliasProp: "fieldChainAlias", wantNamed: "string/ssn",
				note: "dissolve lands on a NAMED annotated type, so the named machinery still applies the format",
			},
			{definition: "Envelope", namedProp: "pointerBasicNamed", aliasProp: "pointerBasicAlias", wantNamed: "string/isbn"},
			{definition: "Envelope", namedProp: "pointerStructNamed", aliasProp: "pointerStructAlias", wantNamed: "string/duration"},
			{definition: "Envelope", namedProp: "sliceElemBasicNamed", aliasProp: "sliceElemBasicAlias", wantNamed: "array<string/isbn>"},
			{definition: "Envelope", namedProp: "sliceElemStructNamed", aliasProp: "sliceElemStructAlias", wantNamed: "array<string/duration>"},
			{definition: "Envelope", namedProp: "mapValueBasicNamed", aliasProp: "mapValueBasicAlias", wantNamed: "map<string/isbn>"},
			{definition: "Envelope", namedProp: "mapValueStructNamed", aliasProp: "mapValueStructAlias", wantNamed: "map<string/duration>"},
			{
				definition: "EnvelopeModeled", namedProp: "modeledBasicNamed", aliasProp: "modeledBasicAlias", wantNamed: "string/isbn",
				note: "the model annotation is an accidental workaround: the alias gets its own definition, where buildDeclAlias:248 applies the format",
			},
			{definition: "EnvelopeModeled", namedProp: "modeledStructNamed", aliasProp: "modeledStructAlias", wantNamed: "string/duration"},
		},

		// No cell in F1 has a legitimate reason to differ: every observed difference is the bug.
		exceptions:    map[string]string{},
		knownBroken:   broken,
		controlBroken: map[string]string{},
	}

	ledger.run(t)
}
