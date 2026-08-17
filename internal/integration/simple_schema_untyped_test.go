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

// A non-body parameter and a response header must carry a type; OAS v2 requires one, so resolving to
// an empty schema there is invalid rather than permissive.
//
// `any` and `json.RawMessage` are the usual sources. They keep the empty schema in a body, where it
// is the right answer, and default to `{type: string}` outside it — with a diagnostic, because the
// fallback is codescan choosing rather than the Go type saying.
func TestSimpleSchemaUntyped(t *testing.T) {
	var diags []codescan.Diagnostic
	doc, err := runScan(&codescan.Options{
		Packages:     []string{"./enhancements/simple-schema-untyped/..."},
		WorkDir:      scantest.FixturesDir(),
		OnDiagnostic: func(d codescan.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	params := map[string]spec2Param{}
	for _, p := range doc.Paths.Paths["/untyped"].Get.Parameters {
		params[p.Name] = spec2Param{Type: p.Type, In: p.In, Ext: p.Extensions}
	}

	t.Run("a non-body parameter defaults to string", func(t *testing.T) {
		for _, name := range []string{"anything", "raw"} {
			require.Contains(t, params, name)
			assert.Equal(t, "string", params[name].Type,
				"%s must carry a type: OAS v2 requires one on a non-body parameter", name)
		}
	})

	t.Run("a response header defaults to string", func(t *testing.T) {
		headers := doc.Responses["untypedResponse"].Headers
		for _, name := range []string{"X-Anything", "X-Raw"} {
			require.Contains(t, headers, name)
			assert.Equal(t, "string", headers[name].Type,
				"%s must carry a type: OAS v2 requires one on a header", name)
		}
	})

	t.Run("no format is invented", func(t *testing.T) {
		// The whole reason for being here is that the Go type said nothing; `binary` would claim octets
		// a percent-encoded position cannot carry, and `byte` a base64 framing nobody applied.
		assert.Empty(t, params["anything"].Ext["format"])
		for _, p := range doc.Paths.Paths["/untyped"].Get.Parameters {
			if p.In == "query" {
				assert.Empty(t, p.Format, "%s must claim no format", p.Name)
			}
		}
	})

	t.Run("x-go-type records what the fallback erased", func(t *testing.T) {
		assert.Equal(t, "any", params["anything"].Ext["x-go-type"])
		assert.Equal(t, "encoding/json.RawMessage", params["raw"].Ext["x-go-type"])
	})

	t.Run("a body keeps the empty schema", func(t *testing.T) {
		// The control: empty is the correct answer where OAS v2 allows it, so the fallback must be
		// scoped to SimpleSchema and not leak into a schema position.
		props := doc.Definitions["Payload"].Properties
		require.NotEmpty(t, props)
		assert.Empty(t, props["anything"].Type, "a body property keeps 'any JSON'")
		assert.Empty(t, props["raw"].Type, "a body property keeps 'any JSON'")
	})

	t.Run("choosing for the author is reported", func(t *testing.T) {
		var n int
		for _, d := range diags {
			if d.Code == grammar.CodeUnderspecifiedInSimpleSchema {
				n++
				assert.Equal(t, codescan.SeverityWarning, d.Severity)
				assert.NotEmpty(t, d.Pos.Filename, "diagnostic must be located")
			}
		}
		assert.Equal(t, 4, n, "two parameters and two headers, each reported once; got %v", diags)
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_simple_schema_untyped.json")
}

type spec2Param struct {
	Type string
	In   string
	Ext  map[string]any
}

// The x-go-type stamp is an extension like any other and must disappear under SkipExtensions, while
// the type it accompanies must not: the type is what makes the spec valid.
func TestSimpleSchemaUntyped_SkipExtensions(t *testing.T) {
	doc, err := runScan(&codescan.Options{
		Packages:       []string{"./enhancements/simple-schema-untyped/..."},
		WorkDir:        scantest.FixturesDir(),
		SkipExtensions: true,
	})
	require.NoError(t, err)

	for _, p := range doc.Paths.Paths["/untyped"].Get.Parameters {
		if p.In != "query" {
			continue
		}
		assert.Equal(t, "string", p.Type, "%s keeps its type", p.Name)
		assert.NotContains(t, p.Extensions, "x-go-type", "%s must carry no extension", p.Name)
	}

	for name, h := range doc.Responses["untypedResponse"].Headers {
		assert.Equal(t, "string", h.Type, "%s keeps its type", name)
		assert.NotContains(t, h.Extensions, "x-go-type", "%s must carry no extension", name)
	}
}
