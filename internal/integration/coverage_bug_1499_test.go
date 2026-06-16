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

// TestCoverage_Bug1499 asserts the EXPECTED resolution of go-swagger issue #1499
// ("not possible to override parameter schema type"). Currently RED.
//
// A `swagger:type` override on a swagger:parameters field whose Go type is a
// struct must make the parameter that simple type (here `string`). The schema
// (model) builder already honours this (#2419/#2184), but the PARAMETERS builder
// ignores it — the param comes out typeless, which is invalid.
func TestCoverage_Bug1499(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1499/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	p := doc.Paths.Paths["/foo"].Get.Parameters[0]
	assert.Equal(t, "bar", p.Name)
	assert.Equal(t, "query", p.In)
	assert.Equal(t, "string", p.Type, "swagger:type must override the param field type (go-swagger#1499)")
}
