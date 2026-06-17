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

// TestQuirk_ModelOverrideSimpleSchema guards the F1/F4 fix against leaking a
// $ref into a SimpleSchema. A swagger:model type with an override (enum /
// strfmt) is referenced by $ref in full-schema mode — but $ref is illegal in
// an OAS-2 SimpleSchema (non-body parameters, response headers), so there the
// override must INLINE instead. The buildNamedType gate is therefore
// (isModel && !simpleSchema): a regression would dissolve the schema to {} via
// the simple-schema $ref safety net, losing the enum/format.
func TestQuirk_ModelOverrideSimpleSchema(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./quirks/model-override-simpleschema/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Query parameter referencing a swagger:model enum: inlined, not $ref'd.
	op := doc.Paths.Paths["/things"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	filter := op.Parameters[0]
	assert.Equal(t, "query", filter.In)
	assert.Equal(t, "string", filter.Type)
	assert.Equal(t, []any{"red", "blue"}, filter.Enum, "enum inlined on the parameter, not lost to a $ref")

	// Response header referencing a swagger:model strfmt: inlined, not $ref'd.
	resp, ok := doc.Responses["thingResponse"]
	require.True(t, ok)
	hdr, ok := resp.Headers["X-Sku"]
	require.True(t, ok)
	assert.Equal(t, "string", hdr.Type)
	assert.Equal(t, "sku", hdr.Format, "strfmt format inlined on the header, not lost to a $ref")
}
