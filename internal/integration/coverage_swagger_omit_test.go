// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const swaggerOmitPkg = "./enhancements/swagger-omit/..."

// runOmit scans the swagger-omit fixture, capturing diagnostics by code.
func runOmit(t *testing.T, defaultAllOf bool) (*oaispec.Swagger, map[grammar.Code][]grammar.Diagnostic) {
	t.Helper()
	byCode := map[grammar.Code][]grammar.Diagnostic{}
	doc, err := codescan.Run(&codescan.Options{
		Packages:              []string{swaggerOmitPkg},
		WorkDir:               scantest.FixturesDir(),
		DefaultAllOfForEmbeds: defaultAllOf,
		OnDiagnostic: func(d grammar.Diagnostic) {
			byCode[d.Code] = append(byCode[d.Code], d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	return doc, byCode
}

// propNames returns the property names of a schema, inlined or composed: an allOf compound is
// flattened across its members so the two renderings can be asserted against one expectation.
func propNames(sch oaispec.Schema) []string {
	var out []string
	var walk func(s oaispec.Schema)
	walk = func(s oaispec.Schema) {
		for name := range s.Properties {
			out = append(out, name)
		}
		for _, m := range s.AllOf {
			walk(m)
		}
	}
	walk(sch)

	return out
}

// TestSwaggerOmit_Inlined locks the pre-filter in the default (inlined) rendering: the listed fields
// are never promoted, whether named on the embed, on the declaration, or through an embed chain.
func TestSwaggerOmit_Inlined(t *testing.T) {
	doc, byCode := runOmit(t, false)

	t.Run("embed-level form drops plain field names of that embed", func(t *testing.T) {
		assert.ElementsMatch(t, []string{"Name", "extra"}, propNames(doc.Definitions["EmbedLevel"]))
	})

	t.Run("declaration-level form takes a dotted path and a bare name", func(t *testing.T) {
		assert.ElementsMatch(t, []string{"Name", "extra"}, propNames(doc.Definitions["DeclLevel"]))
	})

	t.Run("a qualified path reaches through an embed chain", func(t *testing.T) {
		// Nested.Deep is promoted twice over (DeepPath <- Nested <- Inner); Visible and Shallow stay.
		assert.ElementsMatch(t, []string{"Visible", "Shallow", "extra"}, propNames(doc.Definitions["DeepPath"]))
	})

	t.Run("an override keeps the outer declaration, as Go does", func(t *testing.T) {
		// Inlined, the outer field already won: omitting its promoted twin is correctly a no-op here.
		id := doc.Definitions["Decorated"].Properties["ID"]
		assert.Equal(t, oaispec.StringOrArray{"integer"}, id.Type)
		assert.True(t, id.ReadOnly)
		assert.Equal(t, oaispec.StringOrArray{"string"}, doc.Definitions["Retyped"].Properties["ID"].Type)
	})

	t.Run("the go-swagger#1992 shape emits only the wanted field", func(t *testing.T) {
		body := bodyParamOf(t, doc, "/things")
		assert.ElementsMatch(t, []string{"Name"}, propNames(*body),
			"an inline body embedding a shared type carries only what the author kept")
	})

	assert.Empty(t, byCode[grammar.CodeInvalidAnnotation], "the annotation parses cleanly")
	scantest.CompareOrDumpJSON(t, doc, "enhancements_swagger_omit.json")
}

// TestSwaggerOmit_DefaultAllOf locks the headline property: the SAME annotation reads identically
// when the embed is composed instead of inlined — which is what removes the duplicate property in
// `Decorated` and the unsatisfiable integer-AND-string pair in `Retyped`.
func TestSwaggerOmit_DefaultAllOf(t *testing.T) {
	doc, _ := runOmit(t, true)

	t.Run("the omitted field is absent from the composed member", func(t *testing.T) {
		assert.ElementsMatch(t, []string{"Name", "extra"}, propNames(doc.Definitions["EmbedLevel"]))
	})

	t.Run("an override no longer duplicates across members", func(t *testing.T) {
		decorated := doc.Definitions["Decorated"]
		require.Len(t, decorated.AllOf, 2)
		assert.NotContains(t, decorated.AllOf[0].Properties, "ID",
			"the base member must not carry the property the outer field replaces")
		id := decorated.AllOf[1].Properties["ID"]
		assert.True(t, id.ReadOnly)
		assert.Len(t, propNames(decorated), 3, "Created, Name, ID — each exactly once")
	})

	t.Run("a retyped override is no longer unsatisfiable", func(t *testing.T) {
		retyped := doc.Definitions["Retyped"]
		require.Len(t, retyped.AllOf, 2)
		assert.NotContains(t, retyped.AllOf[0].Properties, "ID")
		assert.Equal(t, oaispec.StringOrArray{"string"}, retyped.AllOf[1].Properties["ID"].Type,
			"only the outer declaration's type survives, as in Go")
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_swagger_omit_allof.json")
}

// TestSwaggerOmit_Diagnostics locks the three Hints. `swagger:omit` is the only construct whose
// output depends on a hand-written name the compiler never checks, so an unresolved target must be
// reported or a rename upstream would silently put the field back.
func TestSwaggerOmit_Diagnostics(t *testing.T) {
	_, byCode := runOmit(t, false)

	t.Run("an unresolved target is reported and ignored", func(t *testing.T) {
		hints := byCode[grammar.CodeOmitUnresolved]
		require.Len(t, hints, 1)
		assert.Equal(t, grammar.SeverityHint, hints[0].Severity)
		assert.Contains(t, hints[0].Message, `Base has no field "Createed"`)
		assert.Positive(t, hints[0].Pos.Line)
	})

	t.Run("a target behind a $ref is reported and dropped", func(t *testing.T) {
		hints := byCode[grammar.CodeOmitBehindRef]
		require.Len(t, hints, 1)
		assert.Equal(t, grammar.SeverityHint, hints[0].Severity)
		assert.Contains(t, hints[0].Message, "cannot have a property")
	})

	t.Run("a json:- re-declaration of a promoted field is reported", func(t *testing.T) {
		hints := byCode[grammar.CodeShadowedEmbedField]
		require.Len(t, hints, 1)
		assert.Equal(t, grammar.SeverityHint, hints[0].Severity)
		assert.Contains(t, hints[0].Message, "swagger:omit")
	})
}

// bodyParamOf returns the body parameter schema of the POST operation on path.
func bodyParamOf(t *testing.T, doc *oaispec.Swagger, path string) *oaispec.Schema {
	t.Helper()
	require.NotNil(t, doc.Paths)
	pi, ok := doc.Paths.Paths[path]
	require.True(t, ok, "path %s missing", path)
	require.NotNil(t, pi.Post)
	for _, prm := range pi.Post.Parameters {
		if prm.Schema != nil {
			return prm.Schema
		}
	}
	require.FailNow(t, "no body parameter on "+path)

	return nil
}
