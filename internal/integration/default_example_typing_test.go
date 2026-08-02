// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Witness for the typing of `default:`, `example:` and `enum:` values.
//
// A declaration's comment block is dispatched before its Go type is resolved onto the schema, so
// those three keywords used to be coerced against an empty type and fell back to their raw string
// form — `default: 8080` on a named int became "8080", `enum: 1,2,3` became ["1","2","3"], and a
// JSON array literal became a string holding JSON source. Every other site (field, parameter,
// header, body property) was always correct, because the type is known there when the walk runs.
//
// Each declaration cell is asserted against a field-site control carrying the identical literal, so
// the test states "these must agree" rather than hard-coding a coercion result.
//
// The three keywords are checked together on purpose: they share validations.ParseDefault and the
// same dispatch arms, so a divergence between them is itself a defect.
func TestDefaultExampleTyping(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/default-example-typing/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	controls := doc.Definitions["FieldControls"].Properties

	// decl definition ↔ the field-site control carrying the same literal.
	pairs := []struct {
		decl    string
		control string
	}{
		{"DeclInt", "port"},
		{"DeclNumber", "ratio"},
		{"DeclBool", "flag"},
		{"DeclString", "mode"},
		{"DeclIntSlice", "numbers"},
	}

	for _, p := range pairs {
		t.Run(p.decl, func(t *testing.T) {
			decl, ok := doc.Definitions[p.decl]
			require.True(t, ok, "missing definition %s", p.decl)
			ctl, ok := controls[p.control]
			require.True(t, ok, "missing control property %s", p.control)

			assert.Equal(t, ctl.Default, decl.Default,
				"a declaration's default: must be typed exactly as the same literal on a field")
			assert.Equal(t, ctl.Example, decl.Example,
				"a declaration's example: must be typed exactly as the same literal on a field")
		})
	}

	// enum: shares the defect and the fix. An enum of strings on an integer schema is a spec no
	// validator can satisfy, so this cell is the most consequential of the three keywords.
	t.Run("DeclEnumInt", func(t *testing.T) {
		decl := doc.Definitions["DeclEnumInt"]
		ctl := controls["grade"]

		assert.Equal(t, ctl.Enum, decl.Enum, "a declaration's enum: members must be typed as on a field")
		assert.Equal(t, ctl.Default, decl.Default)
	})

	// Sites that were always correct — pinned so a future change to the recoercion cannot regress
	// them by treating every site as if it needed repair.
	t.Run("already-correct sites", func(t *testing.T) {
		op := doc.Paths.Paths["/typing"].Get
		var queryPort *oaispec.Parameter
		for i := range op.Parameters {
			if op.Parameters[i].Name == "queryPort" {
				queryPort = &op.Parameters[i]
			}
		}
		require.NotNil(t, queryPort, "missing queryPort parameter")
		assert.Equal(t, controls["port"].Default, queryPort.Default, "a non-body parameter types its own default")

		hdr, ok := doc.Responses["typingResponse"].Headers["X-Rate-Limit"]
		require.True(t, ok, "missing response header")
		assert.NotNil(t, hdr.Default)
		assert.IsType(t, controls["port"].Default, hdr.Default, "a response header types its own default")
	})

	// The OTHER sense of "default": a response, not a value. `default:` is not legal in a response
	// block context, so the two mechanisms cannot collide — this pins that they stay apart.
	t.Run("default response is not a default value", func(t *testing.T) {
		op := doc.Paths.Paths["/typing"].Get
		require.NotNil(t, op.Responses.Default, "the default response code must produce responses.default")
		assert.Equal(t, "#/responses/errorResponse", op.Responses.Default.Ref.String())
		assert.Nil(t, op.Responses.Default.Schema, "the default RESPONSE carries no default VALUE")
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_default_example_typing.json")
}
