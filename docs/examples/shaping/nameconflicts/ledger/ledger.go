// SPDX-License-Identifier: Apache-2.0

// Package ledger is the same-package-duplicate corner of the name-conflicts
// witness: two distinct Go types in ONE package both claim `swagger:model
// Entry`. A package cannot own a definition name twice, so codescan keeps one
// (deterministically) and the other falls back to its Go type name.
package ledger

// snippet:dup

// Entry keeps the contested name: the definition is "Entry".
//
// swagger:model Entry
type Entry struct {
	Debit int64 `json:"debit"`
}

// Reversal also asks for "Entry". The name is already taken in this package, so
// it reverts to its Go name, "Reversal", and a diagnostic is raised.
//
// swagger:model Entry
type Reversal struct {
	Credit int64 `json:"credit"`
}

// endsnippet:dup
