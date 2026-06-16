// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestQuirk_NameKeywordUniversal verifies the fix for quirk G2
// (doc-site-quirks.md): the `name:` keyword now renames a field at every
// site — swagger:model properties and interface methods included — not just
// swagger:parameters / swagger:response fields. The legacy swagger:name
// annotation is still honoured, and `name:` takes precedence over it.
//
// Precedence: name: keyword > swagger:name > json tag > Go field name.
func TestQuirk_NameKeywordUniversal(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:     []string{"./quirks/name-keyword-universal/..."},
		WorkDir:      scantest.FixturesDir(),
		ScanModels:   true,
		OnDiagnostic: func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	acct := doc.Definitions["Account"].Properties
	require.NotNil(t, acct)

	// name: keyword renames a model property, overriding the json tag.
	require.Contains(t, acct, "balance")
	assert.NotContains(t, acct, "bal")
	assert.Equal(t, "Bal", acct["balance"].Extensions["x-go-name"], "Go name retained as x-go-name")

	// name: wins over a co-located swagger:name annotation.
	require.Contains(t, acct, "currencyCode")
	assert.NotContains(t, acct, "legacyCurrency")
	assert.NotContains(t, acct, "currency")

	// legacy swagger:name on its own is still honoured.
	require.Contains(t, acct, "accountType")
	assert.NotContains(t, acct, "Kind")

	// no override → json tag retained.
	require.Contains(t, acct, "plain")

	// name: works on an interface-method property; swagger:name still does too.
	tok := doc.Definitions["Token"].Properties
	require.NotNil(t, tok)
	require.Contains(t, tok, "issuedAt")
	require.Contains(t, tok, "subject")

	// regression guard: name: still renames a query parameter field.
	op := doc.Paths.Paths["/accounts"].Post
	require.NotNil(t, op)
	var renamed, region bool
	for _, p := range op.Parameters {
		switch p.Name {
		case "account_id":
			renamed = true
		case "Region":
			region = true // swagger:name was dropped → Go-derived name kept
		}
	}
	assert.True(t, renamed, "name: renamed the query parameter to account_id")
	assert.True(t, region, "swagger:name was dropped on the param field; Go name retained")
	assert.NotContains(t, paramNames(op.Parameters), "region_code", "swagger:name must not rename a param")

	// G2 diagnostics: swagger:name in a param / response-header context is
	// inert and now draws a CodeContextInvalid warning pointing at `name:`.
	var paramWarn, headerWarn bool
	for _, d := range diags {
		if d.Code != grammar.CodeContextInvalid || !strings.Contains(d.Message, "swagger:name") {
			continue
		}
		switch {
		case strings.Contains(d.Message, "parameter"):
			paramWarn = true
		case strings.Contains(d.Message, "response header"):
			headerWarn = true
		}
	}
	assert.True(t, paramWarn, "swagger:name on a parameter field must warn")
	assert.True(t, headerWarn, "swagger:name on a response header field must warn")

	scantest.CompareOrDumpJSON(t, doc, "quirk_name_keyword_universal.json")
}

func paramNames(params []oaispec.Parameter) []string {
	out := make([]string, len(params))
	for i := range params {
		out[i] = params[i].Name
	}
	return out
}
