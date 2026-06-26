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

// TestCoverage_Bug1595 locks the fix for go-swagger issue #1595 ("multi-line/block instead of
// single-line comments").
//
// A swagger:operation written in a /* */ block comment (a legal Go doc comment) used to fail the
// whole scan with a YAML parse error ("did not find expected alphabetic or numeric character"): the
// path-annotation parser reshaped the block comment line-by-line and let the closing */ leak into
// the YAML body, where it reads as a YAML alias indicator.
//
// Both the flush-left block style and the idiomatic `*`-decorated style must parse like their //
// equivalent — the path with a 200 response — without erroring.
func TestCoverage_Bug1595(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1595/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err, "a /* */ block-comment annotation must not fail the scan (go-swagger#1595)")
	require.NotNil(t, doc.Paths)

	// Flush-left block comment.
	op := doc.Paths.Paths["/products"].Get
	require.NotNil(t, op, "the flush-left block-comment swagger:operation is discovered")
	assert.Equal(t, "displayProduct", op.ID)
	require.NotNil(t, op.Responses)
	assert.Contains(t, op.Responses.StatusCodeResponses, 200,
		"the YAML body parsed: 200 response present")

	// `*`-decorated block comment (the godoc continuation style).
	starOp := doc.Paths.Paths["/widgets"].Get
	require.NotNil(t, starOp, "the *-decorated block-comment swagger:operation is discovered")
	assert.Equal(t, "displayWidget", starOp.ID)
	require.NotNil(t, starOp.Responses)
	assert.Contains(t, starOp.Responses.StatusCodeResponses, 200,
		"the *-decorated YAML body parsed: 200 response present")
}
