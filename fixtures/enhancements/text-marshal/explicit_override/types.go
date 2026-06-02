// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package explicit_override exercises the precedence rule inside
// buildFromTextMarshal: an explicit `swagger:strfmt` annotation on a
// UUID-named TextMarshaler wins over the implicit UUID heuristic.
//
// Under the old order the heuristic ran first; this `UUID` would have
// been emitted as `{string, uuid}` even though the user explicitly
// annotated it as a `date`. The new order runs the classifier first.
package explicit_override

// UUID is a TextMarshaler whose name matches the case-insensitive
// UUID heuristic but carries an explicit `swagger:strfmt date`
// override. Expected output: `{string, date}` — not `{string, uuid}`.
//
// swagger:strfmt date
type UUID [16]byte

// MarshalText renders the UUID as text.
func (u UUID) MarshalText() ([]byte, error) { return nil, nil }

// UnmarshalText parses the UUID from text.
func (u *UUID) UnmarshalText([]byte) error { return nil }

// OverrideContainer references UUID so the schema builder walks
// the override path.
//
// swagger:model override_container
type OverrideContainer struct {
	// required: true
	ID UUID `json:"id"`
}
