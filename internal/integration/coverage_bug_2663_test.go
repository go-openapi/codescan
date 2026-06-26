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

// TestCoverage_Bug2663 documents the recommended idiom for go-swagger issue #2663 ("How to document
// a body parameter in a POST request?").
//
// The answer is a doc-only one: a body parameter is a SINGLE parameter whose Go type is the body
// schema.
// Declare exactly ONE field carrying `in: body`; give it a named type so the schema is emitted as a
// reusable $ref.
//
// This is also the contrast against the proposals floated on the issue:
//   - marking each scalar field `in: body` emits a SEPARATE body parameter per
//     field, which is invalid Swagger 2.0 (an operation may have at most one
//     body parameter);
//   - an inline anonymous Body struct works but yields a non-reusable inline
//     schema;
//   - unexported body fields emit no properties at all.
//
// 📖 Need doc: document the one-field/named-type body-parameter idiom.
func TestCoverage_Bug2663(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2663/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/login"].Post
	require.NotNil(t, op)

	// Exactly ONE body parameter — the crux of the issue (not one per field).
	require.Len(t, op.Parameters, 1)
	body := op.Parameters[0]
	assert.Equal(t, "body", body.In)
	assert.True(t, body.Required)
	require.NotNil(t, body.Schema)
	assert.Equal(t, "#/definitions/AppUserCredentials", body.Schema.Ref.String(),
		"a named body type yields a reusable $ref, not an inline schema")
	assert.Empty(t, body.Schema.Type, "a $ref body schema must not carry a sibling type")

	// The body schema's own shape, including the named-string field type.
	creds := doc.Definitions["AppUserCredentials"]
	require.Contains(t, creds.Properties, "username")
	require.Contains(t, creds.Properties, "password")
	username := creds.Properties["username"]
	password := creds.Properties["password"]
	assert.Equal(t, "#/definitions/CPF", username.Ref.String(),
		"the named string type CPF is preserved as its own definition")
	assert.Equal(t, []string{"string"}, []string(password.Type))

	cpf := doc.Definitions["CPF"]
	assert.Equal(t, []string{"string"}, []string(cpf.Type))

	scantest.CompareOrDumpJSON(t, doc, "bugs_2663_schema.json")
}
