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

// TestCoverage_MetaSecurityDefsDuplicateKeys — locks the post-Q28
// behaviour of the swagger:meta SecurityDefinitions raw block.
//
// Two independent wires meet here:
//
//   - Grammar lexer: the SecurityDefinitions body must reach the
//     downstream YAML decoder with per-line structural indentation
//     intact (Raw view, not Text view). Without that, both api_key's
//     and oauth2's children collapse into one flat mapping at the
//     top level, the spec decode aborts with a JSON unmarshal error,
//     and the canonical classification fixture stops scanning
//     standalone.
//
//   - YAML sub-parser: duplicate mapping keys are silently last-wins
//     dedupe'd before the strict yaml.v3 decoder sees them. The
//     `api_key` block here has `type:` twice on purpose so the
//     dedupe layer is exercised in the captured spec.
//
// Diagnostic emission on duplicates is deferred to the yaml-library
// swap (see the forthcoming-features roadmap).
func TestCoverage_MetaSecurityDefsDuplicateKeys(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/meta-securitydefs-duplicate-keys/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	require.Contains(t, doc.SecurityDefinitions, "api_key",
		"api_key scheme must survive the lexer-indent + dedupe pipeline")
	require.Contains(t, doc.SecurityDefinitions, "oauth2",
		"oauth2 scheme must survive the lexer-indent + dedupe pipeline")

	apiKey := doc.SecurityDefinitions["api_key"]
	require.NotNil(t, apiKey)
	assert.Equal(t, "apiKey", apiKey.Type,
		"api_key.type: last-wins dedupe must keep the second occurrence (same value here)")
	assert.Equal(t, "KEY", apiKey.Name)
	assert.Equal(t, "header", apiKey.In)

	oauth2 := doc.SecurityDefinitions["oauth2"]
	require.NotNil(t, oauth2)
	assert.Equal(t, "oauth2", oauth2.Type)
	assert.Equal(t, "accessCode", oauth2.Flow)
	assert.Equal(t, "/oauth2/auth", oauth2.AuthorizationURL)
	assert.Equal(t, "/oauth2/token", oauth2.TokenURL)
	require.Contains(t, oauth2.Scopes, "bla1")
	require.Contains(t, oauth2.Scopes, "bla2")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_meta_securitydefs_duplicate_keys.json")
}
