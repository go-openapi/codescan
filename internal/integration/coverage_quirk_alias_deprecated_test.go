// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestQuirk_AliasDeprecated locks the F8 deprecation of swagger:alias (doc-site-quirks.md).
// swagger:alias used to force a named primitive to inline ({type:string}) instead of the $ref a
// named type otherwise gets.
//
// It is now an empty sink: it emits a validate.deprecated diagnostic and has no effect on output,
// so the type resolves through default handling (here, a $ref to its definition under
// swagger:model).
func TestQuirk_AliasDeprecated(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:     []string{"./quirks/alias-deprecated/..."},
		WorkDir:      scantest.FixturesDir(),
		ScanModels:   true,
		OnDiagnostic: func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// A deprecation diagnostic was emitted for swagger:alias.
	var deprecated bool
	for _, d := range diags {
		if d.Code == grammar.CodeDeprecated {
			assert.Contains(t, d.Message, "swagger:alias")
			deprecated = true
		}
	}
	assert.True(t, deprecated, "swagger:alias must emit a deprecation diagnostic")

	// No effect on output: the field resolves to a $ref (default handling), not the inline
	// {type:string} swagger:alias used to force.
	contact := doc.Definitions["Account"].Properties["contact"]
	ref := contact.Ref
	assert.Equal(t, "#/definitions/Email", ref.String())
	assert.Empty(t, contact.Type, "must not be an inlined scalar")
}
