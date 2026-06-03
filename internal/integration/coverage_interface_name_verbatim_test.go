// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_InterfaceNameVerbatim pins the Q9 contract: when the
// user provides `swagger:name X` on an interface method, X is
// emitted verbatim — the auto-jsonify mangler is bypassed.
//
// The default-path method (no swagger:name) is asserted alongside to
// triangulate against the mangler's lowercase-first-camelCase
// behaviour: any future regression that runs the mangler over
// user-provided names would change the four override entries.
//
// See observed-quirks Q9 (RESOLVED, Stream M) and
// internal/builders/schema/README.md §interface-naming.
func TestCoverage_InterfaceNameVerbatim(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/interface-name-verbatim/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	require.Contains(t, doc.Definitions, "NameOverrides")
	props := doc.Definitions["NameOverrides"].Properties

	// Default path — mangler runs, lowercase-first camelCase.
	require.Contains(t, props, "userIdentifier", "default path must auto-jsonify")

	// Each swagger:name override must reach the spec verbatim. If
	// any of these fails, the mangler is running over user input —
	// the exact regression Q9's resolution forbids.
	assert.Contains(t, props, "UserIdentifier", "PascalCase override must stay PascalCase")
	assert.Contains(t, props, "user_identifier", "snake_case override must stay snake_case")
	assert.Contains(t, props, "USER_ID", "SCREAMING_CASE override must stay SCREAMING_CASE")
	assert.Contains(t, props, "X-Custom-Header", "hyphenated override must stay hyphenated")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_interface_name_verbatim.json")
}
