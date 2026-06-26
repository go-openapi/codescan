// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package godoclink

import (
	"testing"

	"github.com/go-openapi/swag/mangling"
	"github.com/go-openapi/testify/v2/assert"
)

func TestClean(t *testing.T) {
	m := mangling.NewNameMangler()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"no brackets", "A plain sentence with no links.", "A plain sentence with no links."},

		// doc-link spans: brackets stripped + leaf humanized.
		{"single uppercase mid-sentence", "Owned by a [Person] here.", "Owned by a person here."},
		{"dotted leaf", "Tracked in [inventory.Ledger] now.", "Tracked in ledger now."},
		{"type.field leaf", "See [Order.CustName] for it.", "See cust name for it."},
		{"pointer span", "Wraps a [*time.Time] value.", "Wraps a time value."},

		// sentence-initial substitution is restored to sentence case.
		{"leading link titleized", "[Widget] does things.", "Widget does things."},
		{"leading dotted titleized", "[inventory.Ledger] holds rows.", "Ledger holds rows."},
		{"leading after spaces", "  [Person] owns it.", "  Person owns it."},

		// reference-style link definition lines are dropped.
		{
			"ref def line dropped",
			"A widget.\n\n[the spec]: https://example.com/spec",
			"A widget.",
		},
		{
			"ref def interior line folded",
			"Intro line.\n\n[ref]: https://x.test\n\nOutro line.",
			"Intro line.\n\nOutro line.",
		},

		// conservative recognizer: ordinary prose brackets are left intact.
		{"empty brackets byte", "A [] field of type []byte.", "A [] field of type []byte."},
		{"digit index", "Element [0] of the slice.", "Element [0] of the slice."},
		{"multiword brackets", "As noted [see notes] above.", "As noted [see notes] above."},
		{"bare lowercase", "The [id] field is required.", "The [id] field is required."},

		// multiple links in one line.
		{
			"multiple links",
			"[Widget] links to a [Person] and an [Order].",
			"Widget links to a person and an order.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Clean(tc.in, Options{Mangler: &m}))
		})
	}
}

func TestCleanLeavesInitialismCasing(t *testing.T) {
	m := mangling.NewNameMangler()
	// ToHumanNameLower keeps recognized initialisms cased; CleanGoDoc must not re-case them
	// mid-sentence.
	assert.Equal(t, "Serves over an HTTP server.", Clean("Serves over an [HTTPServer].", Options{Mangler: &m}))
}
