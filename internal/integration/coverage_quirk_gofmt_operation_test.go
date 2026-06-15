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

// TestQuirk_GofmtOperationYAML is the operations counterpart of quirk F7
// (TestQuirk_GofmtMetaYAML). A swagger:operation YAML body in its
// gofmt-canonical shape — column-0 top-level keys rendered as prose (one
// leading space), their value blocks rendered as tab-indented code blocks,
// separated by blank "//" lines — must parse identically to the space-indented
// form. Previously the tab indentation reached the YAML sub-parser, which
// rejects tabs, and the nesting collapsed: `responses` degraded to a single
// `default` with an empty description and the vendor extension became `null`.
//
// F7 fixed the swagger:meta case (a uniformly tab-prefixed body); this asserts
// the operation case, whose body interleaves 1-space prose keys with
// tab-prefixed value blocks. The dedent now expands leading tabs to spaces
// before stripping the common indent.
func TestQuirk_GofmtOperationYAML(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./quirks/gofmt-operation/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/aws"].Get
	require.NotNil(t, op)

	// The operation's own responses parsed (not collapsed to a bare default).
	r200, ok := op.Responses.StatusCodeResponses[200]
	require.True(t, ok, "the 200 response must survive the gofmt tab indentation")
	assert.Equal(t, "ok", r200.Description)

	// The vendor extension is preserved with its nested map (incl. responses),
	// not dropped to null.
	ext, ok := op.Extensions["x-amazon-apigateway-integration"]
	require.True(t, ok)
	m, ok := ext.(map[string]any)
	require.True(t, ok, "the extension value must be a map, not null")
	assert.Equal(t, "http_proxy", m["type"])
	assert.Equal(t, "https://proxy-url.com", m["uri"])
	assert.Contains(t, m, "responses", "the nested responses key inside the extension survives")

	scantest.CompareOrDumpJSON(t, doc, "quirk_gofmt_operation.json")
}
