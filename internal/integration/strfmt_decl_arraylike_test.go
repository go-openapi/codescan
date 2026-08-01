// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import "testing"

// Witness for the element-driven items-vs-whole rule on a `swagger:strfmt` over an array or slice,
// at BOTH the declaration site and a field site, for named and alias halves alike.
//
// Two things were wrong before. The declaration switch had arms for struct and basic only, so a
// model sequence published its definition with the format dropped — and nothing downstream
// compensates, because the refModel gate skips the inline classifiers on the assumption the
// declaration already applied it. And the items-vs-whole decision was a two-name allowlist
// (`byte`, `bsonobjectid`) that stood in for the real question: both are formats for a byte
// sequence. Keying on the element type generalises to `uuid` over `[16]byte`, to `ulid`, and to
// rune sequences, with no list to extend.
//
// See strfmt_symmetry_harness_test.go and internal/builders/schema/README.md#aliases.
func TestStrfmtDeclArrayLike(t *testing.T) {
	ledger := symmetryLedger{
		pkg:          "enhancements/strfmt-decl-arraylike",
		goldenPrefix: "strfmt_decl_arraylike",
		cells: []symmetryCell{
			// Declaration site — a byte sequence takes the format on the schema.
			{
				namedProp: "IDNamedModeled", aliasProp: "IDAliasModeled",
				wantNamed: "string/uuid",
				note:      "[16]byte is a byte sequence, so uuid describes the whole value",
			},
			{
				namedProp: "ULIDNamedModeled", aliasProp: "ULIDAliasModeled",
				wantNamed: "string/ulid",
				note:      "generalises past the old byte/bsonobjectid allowlist",
			},
			{
				namedProp: "RunesNamedModeled", aliasProp: "RunesAliasModeled",
				wantNamed: "string/password",
				note:      "rune sequences are string-like too",
			},

			// Declaration site — a string slice keeps the format on its items.
			{
				namedProp: "EmailsNamedModeled", aliasProp: "EmailsAliasModeled",
				wantNamed: "array<string/email>",
				note:      "element is a string, so the format describes each element",
			},

			// Field site — same rule, reached through the inline classifier rather than the decl.
			{
				definition: "Envelope", namedProp: "fieldIdNamed", aliasProp: "fieldIdAlias",
				wantNamed: "string/uuid",
			},
		},

		exceptions:    map[string]string{},
		knownBroken:   map[string]string{},
		controlBroken: map[string]string{},
	}

	ledger.run(t)
}
