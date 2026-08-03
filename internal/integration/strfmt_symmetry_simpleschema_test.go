// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/require"
)

// F3 of the strfmt dispatch-symmetry matrix: the SimpleSchema locations — non-body parameters and
// response headers — where OAS v2 forbids `$ref`, so the builder runs with `simpleSchema` set and
// the `refModel` gate flips (`schema.go:475`).
//
// See strfmt_symmetry_harness_test.go for how the ledger reads.
func TestStrfmtSymmetrySimpleSchema(t *testing.T) {
	ledger := symmetryLedger{
		pkg:          "enhancements/strfmt-symmetry-simpleschema",
		goldenPrefix: "strfmt_symmetry_simpleschema",
		cells: []symmetryCell{
			{
				namedProp: "queryBasicNamed", aliasProp: "queryBasicAlias",
				wantNamed:  "string/isbn",
				signatures: queryParamSignatures("queryBasicNamed", "queryBasicAlias"),
			},
			{
				namedProp: "querySliceNamed", aliasProp: "querySliceAlias",
				wantNamed:  "string/byte",
				signatures: queryParamSignatures("querySliceNamed", "querySliceAlias"),
			},
			{
				namedProp: "headerBasicNamed", aliasProp: "headerBasicAlias",
				wantNamed:  "string/isbn",
				signatures: responseHeaderSignatures("headerBasicNamed", "headerBasicAlias"),
			},
			{
				namedProp: "headerSliceNamed", aliasProp: "headerSliceAlias",
				wantNamed:  "string/byte",
				signatures: responseHeaderSignatures("headerSliceNamed", "headerSliceAlias"),
			},
		},

		exceptions:    map[string]string{},
		knownBroken:   map[string]string{},
		controlBroken: map[string]string{},
	}

	ledger.run(t)
}

// operationParams returns the parameters of the fixture's single operation, keyed by name.
func operationParams(t *testing.T, doc *oaispec.Swagger) map[string]oaispec.Parameter {
	t.Helper()

	require.NotNil(t, doc.Paths, "fixture must produce paths")
	path, ok := doc.Paths.Paths["/simple"]
	require.True(t, ok, "missing path /simple")
	require.NotNil(t, path.Get, "missing GET operation on /simple")

	out := make(map[string]oaispec.Parameter, len(path.Get.Parameters))
	for _, p := range path.Get.Parameters {
		out[p.Name] = p
	}
	return out
}

// queryParamSignatures locates two query parameters and renders their SimpleSchema signatures.
func queryParamSignatures(named, alias string) func(*testing.T, *oaispec.Swagger) (string, string) {
	return func(t *testing.T, doc *oaispec.Swagger) (string, string) {
		t.Helper()

		params := operationParams(t, doc)
		namedParam, okNamed := params[named]
		require.True(t, okNamed, "missing query parameter %s", named)
		aliasParam, okAlias := params[alias]
		require.True(t, okAlias, "missing query parameter %s", alias)

		return simpleSignature(namedParam.Type, namedParam.Format, namedParam.Items),
			simpleSignature(aliasParam.Type, aliasParam.Format, aliasParam.Items)
	}
}

// responseHeaderSignatures locates two headers on the shared response definition.
//
// The operation's 200 only carries `$ref: #/responses/simpleResponse`; the headers themselves live
// in the top-level responses section.
func responseHeaderSignatures(named, alias string) func(*testing.T, *oaispec.Swagger) (string, string) {
	return func(t *testing.T, doc *oaispec.Swagger) (string, string) {
		t.Helper()

		resp, ok := doc.Responses["simpleResponse"]
		require.True(t, ok, "missing response definition simpleResponse")

		namedHeader, okNamed := resp.Headers[named]
		require.True(t, okNamed, "missing response header %s", named)
		aliasHeader, okAlias := resp.Headers[alias]
		require.True(t, okAlias, "missing response header %s", alias)

		return simpleSignature(namedHeader.Type, namedHeader.Format, namedHeader.Items),
			simpleSignature(aliasHeader.Type, aliasHeader.Format, aliasHeader.Items)
	}
}
