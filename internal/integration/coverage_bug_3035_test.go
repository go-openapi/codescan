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

// TestCoverage_Bug3035 locks the fix for go-swagger issue #3035 ("Example spec for swagger:response
// does not produce example output").
//
// A swagger:response whose body is an anonymous inline struct used to emit only the response
// description with no schema, dropping the field-level Example/Required and property descriptions
// entirely.
//
// The fix produces a full object schema.
// This test asserts the schema and, crucially, the per-field example — the subject of the issue.
//
// Known, intentional delta vs the reporter's hand-written expected YAML: the body field's leading
// prose ("The error message") is NOT propagated to the schema-level description.
// No code path in codescan surfaces a body field's prose as the schema description (verified
// against all response goldens), and an added blank line after that prose does not change it.
//
// We lock the current behaviour; surfacing it would be a separate enhancement.
func TestCoverage_Bug3035(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/3035/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	resp, ok := doc.Responses["validationError"]
	require.True(t, ok, "the swagger:response must be present")
	require.NotNil(t, resp.Schema, "the inline body struct must produce a schema (the regression: schema was missing)")

	sch := resp.Schema
	assert.Equal(t, "object", sch.Type[0])
	assert.Equal(t, []string{"Message"}, sch.Required)

	require.Contains(t, sch.Properties, "Message")
	msg := sch.Properties["Message"]
	assert.Equal(t, "string", msg.Type[0])
	assert.Equal(t, "The validation message", msg.Description)
	assert.Equal(t, "Expected type int", msg.Example, "the field example is the subject of the issue")

	require.Contains(t, sch.Properties, "FieldName")
	assert.Equal(t, "An optional field name to which this validation applies", sch.Properties["FieldName"].Description)

	// Documented delta: the body field's leading prose is not surfaced.
	assert.Empty(t, sch.Description, "body-field prose is not propagated to the schema description (documented behaviour)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_3035_schema.json")
}
