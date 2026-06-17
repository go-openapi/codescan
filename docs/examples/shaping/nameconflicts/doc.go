// SPDX-License-Identifier: Apache-2.0

// Package nameconflicts is the witness for the "Resolving $ref name conflicts"
// how-to. nameconflicts_test.go scans this package tree and writes the golden
// definitions the guide renders, so the documentation can never drift from the
// scanner's real deconfliction output.
//
// The tree exercises three situations the name-identity reduce stage handles:
//
//   - billing.Account vs identity.Account — two packages, same leaf: each is
//     qualified with its package segment (BillingAccount / IdentityAccount).
//   - ledger.Entry declared twice in one package — first wins, the loser
//     reverts to its Go name (Adjustment).
//   - a cross-package type-name keyword (Bag's additionalProperties) naming a
//     unique leaf in another package (catalog.Widget) resolves to a $ref.
package nameconflicts

import (
	"github.com/go-openapi/codescan/docs/examples/shaping/nameconflicts/billing"
	"github.com/go-openapi/codescan/docs/examples/shaping/nameconflicts/identity"
	"github.com/go-openapi/codescan/docs/examples/shaping/nameconflicts/ledger"
)

// snippet:dashboard

// Dashboard references the same-named Account from two packages plus both
// ledger entries, forcing all of them to be discovered together. The two
// Accounts collide on the short name and are deconflicted by package segment;
// the refs below point at the resolved names, never a bare "Account".
//
// swagger:model Dashboard
type Dashboard struct {
	Billing  billing.Account  `json:"billing"`
	Identity identity.Account `json:"identity"`
	// the ledger entry that kept the name "Entry"
	Primary ledger.Entry `json:"primary"`
	// the duplicate that reverted to its Go name "Reversal"
	Secondary ledger.Reversal `json:"secondary"`
}

// endsnippet:dashboard

// snippet:bag

// Bag is an open object whose additional properties are catalog.Widget,
// named by the bare leaf "Widget" — resolved cross-package because that leaf is
// unique across the scanned model set.
//
// swagger:model Bag
// swagger:additionalProperties Widget
type Bag struct {
	ID string `json:"id"`
}

// endsnippet:bag
