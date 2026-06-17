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

// TestCoverage_SecurityYAMLIdiom locks the recommended form of a global
// Security requirements block: idiomatic OpenAPI 2.0 YAML — a sequence of
// Security Requirement Objects, scopes in block style. The captured spec is
// identical to what a real YAML parse of the same source produces, which is
// now literally true: the security sub-parser (internal/parsers/security)
// decodes the body as YAML rather than hand-parsing it.
//
//	Security:
//	  - oauth:                 # ┐ one requirement object:
//	      - read               # │   oauth (with scopes)
//	      - write              # │   AND
//	    api_key: []            # ┘   api_key
//	  - basic: []              #   OR this alternative
//
// Doc-facing companion to the per-issue witnesses #2403 (sequence marker /
// inline scopes), #2479 (explicit empty opt-out) and #2294 (multi-key AND
// grouping): the positive demonstration that those forms are just YAML.
func TestCoverage_SecurityYAMLIdiom(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/security-yaml-idiom/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// (oauth AND api_key) OR basic → two requirement objects.
	require.Len(t, doc.Security, 2, "two OR-alternatives")

	and := doc.Security[0]
	require.Contains(t, and, "oauth", "first alternative ANDs oauth …")
	require.Contains(t, and, "api_key", "… with api_key")
	assert.Equal(t, []string{"read", "write"}, and["oauth"], "flow-style scope list parses")
	assert.Empty(t, and["api_key"], "empty flow list `[]` is empty scopes")

	or := doc.Security[1]
	require.Contains(t, or, "basic", "second alternative is basic alone")
	assert.Empty(t, or["basic"])

	// The schemes are also defined (proves the YAML-bodied SecurityDefinitions
	// path and the requirements path agree on the same scheme names).
	require.Contains(t, doc.SecurityDefinitions, "oauth")
	require.Contains(t, doc.SecurityDefinitions, "api_key")
	require.Contains(t, doc.SecurityDefinitions, "basic")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_security_yaml_idiom.json")
}
