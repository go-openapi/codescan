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

// TestCoverage_Bug2002 locks go-swagger issue #2002 ("generate spec fails with
// invalid type error"): a swagger:response body field whose type lives in
// another package resolves to a $ref definition. The reporter's "unsupported
// type 'invalid type'" came from the old GO111MODULE=off go/loader; the
// module-aware go/packages loader resolves the cross-package type.
func TestCoverage_Bug2002(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2002/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	resp, ok := doc.Responses["foobarResponse"]
	require.True(t, ok)
	require.NotNil(t, resp.Schema)
	assert.Equal(t, "#/definitions/FooBarResponse", resp.Schema.Ref.String())
	_, ok = doc.Definitions["FooBarResponse"]
	assert.True(t, ok, "the cross-package body type is emitted as a definition")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2002_schema.json")
}
