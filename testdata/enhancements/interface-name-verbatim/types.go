// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package interface_name_verbatim witnesses Q9's resolved post-Stream-M
// contract for interface-method property naming.
//
// Interface methods auto-jsonify via the swag/mangling NameMangler
// (ToJSONName — lowercase-first camelCase) because Go's encoding/json
// can't naturally marshal embedded interfaces without a custom
// marshaler, so the scanner invents a sensible default JSON name.
//
// When the user provides `swagger:name X`, X MUST be taken
// verbatim: the mangler is bypassed entirely. This fixture pins
// that contract with non-camelCase user-provided names so any
// future regression (re-running the mangler on user input) turns
// the integration golden red.
//
// See observed-quirks Q9 (RESOLVED, Stream M) and
// internal/builders/schema/README.md §interface-naming.
package interface_name_verbatim

// NameOverrides exercises every shape of user-provided override we
// want to lock against future re-mangling:
//
//   - PascalCase user input must stay PascalCase.
//   - snake_case user input must stay snake_case.
//   - SCREAMING_CASE user input must stay SCREAMING_CASE.
//   - A name that the mangler would rewrite (e.g. an all-caps
//     acronym at the start) must not be touched.
//
// swagger:model NameOverrides
type NameOverrides interface {
	// Default — no swagger:name. The mangler should produce
	// "userIdentifier" (lowercase-first camelCase).
	UserIdentifier() string

	// swagger:name UserIdentifier
	//
	// Override to the verbatim Go name. Re-mangling would lowercase
	// to "userIdentifier" — the witness asserts on the capital U.
	GoNameVerbatim() string

	// swagger:name user_identifier
	//
	// Snake-case user input. Re-mangling would camelCase to
	// "userIdentifier".
	SnakeCaseOverride() string

	// swagger:name USER_ID
	//
	// All-caps user input. Re-mangling would lowercase to "userId"
	// or "user_id".
	ScreamingCaseOverride() string

	// swagger:name X-Custom-Header
	//
	// Hyphenated user input. Re-mangling would strip hyphens.
	HyphenatedOverride() string
}
