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

// TestCoverage_Bug2403 locks the resolution of go-swagger issue #2403 ("go swagger security
// auth0").
//
// A global `Security:` requirement in swagger:meta written as a YAML sequence
// (`- auth0: []`) must parse into a requirement naming `auth0` with empty scopes.
// Before the fix the scanner glued the `- ` sequence marker onto the key,
// producing a requirement keyed by the literal string "- auth0" with a bogus
// scope `"[]"`. The fix lives in internal/parsers/security (stripSequenceMarker
// + parseScopes).
func TestCoverage_Bug2403(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2403/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	require.Len(t, doc.Security, 1, "one global security requirement")
	req := doc.Security[0]
	_, badKey := req["- auth0"]
	assert.False(t, badKey, "the YAML sequence marker must not be glued onto the key (go-swagger#2403)")
	scopes, ok := req["auth0"]
	require.True(t, ok, "the requirement must be keyed by the scheme name `auth0`")
	assert.Empty(t, scopes, "empty scopes `[]` must be an empty list, not `[\"[]\"]`")
}
