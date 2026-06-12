// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug3214 locks the fix for go-swagger issue #3214
// ("Incomplete parsing of referenced typed primitives"). A struct field
// that references a named primitive type whose doc comment carries both
// prose and an inline `enum:` annotation must:
//
//   - parse the `enum:` declaration into enum values on the referenced
//     type's definition, and
//   - preserve the prose untouched, never folding the `enum:` line into
//     the title or description.
//
// The historical bug folded the `enum:` declaration line into the
// description. This test asserts the parsed shape directly and captures
// the full output in a golden as a regression lock.
func TestCoverage_Bug3214(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3214/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// State: single-line prose → title, with enum parsed and no leakage.
	state, ok := doc.Definitions["State"]
	require.True(t, ok, "State must be emitted as its own definition")
	assert.Equal(t, "string", state.Type[0])
	assert.Equal(t, "State represents the state of an order.", state.Title)
	assert.Equal(t, []any{"created", "processed"}, state.Enum)
	assert.NotContains(t, state.Title, "enum:", "the enum line must not leak into the title")
	assert.NotContains(t, state.Description, "enum:", "the enum line must not leak into the description")

	// Currency: multi-paragraph prose → title + description, enum parsed
	// and the description paragraph survives intact.
	currency, ok := doc.Definitions["Currency"]
	require.True(t, ok, "Currency must be emitted as its own definition")
	assert.Equal(t, "string", currency.Type[0])
	assert.Equal(t, "Currency is the billing currency of an order.", currency.Title)
	assert.True(t, strings.Contains(currency.Description, "ISO-4217"),
		"the prose description paragraph must be preserved")
	assert.Equal(t, []any{"USD", "EUR", "JPY"}, currency.Enum)
	assert.NotContains(t, currency.Description, "enum:", "the enum line must not leak into the description")

	scantest.CompareOrDumpJSON(t, doc, "bugs_3214_schema.json")
}
