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

// TestCoverage_Bug1726 locks the fix for go-swagger#1726 ("How to annotate lists?"): markdown-style
// bullet lists must be identified like the YAML-style dash form.
//
// The lexer normalises `*`/`+` bullets to the canonical `- `, so the relaxation propagates
// uniformly — both prose descriptions and value lists (consumes/produces, via Property.AsList)
// recognise them.
//
// `* item` / `+ item` are normalised to `- item` (matching gofmt's own rewrite of doc-comment
// bullets), so the asserted descriptions carry the dash form.
func TestCoverage_Bug1726(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1726/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	// Guard rail: dash bullets are preserved as a list.
	widget := doc.Definitions["Widget"]
	assert.Contains(t, widget.Description, "- red")
	assert.Contains(t, widget.Description, "- green")

	// Asterisk bullets: identified as a list, normalised to the dash form (not stripped to a run-on
	// line, and not left as a stray `*`).
	gadget := doc.Definitions["Gadget"]
	assert.Contains(t, gadget.Description, "- fast",
		"asterisk bullets must be identified as a list (go-swagger#1726)")
	assert.Contains(t, gadget.Description, "- cheap")
	assert.NotContains(t, gadget.Description, "* fast",
		"the `* ` marker is normalised to `- `")

	// Plus bullets: same treatment.
	gizmo := doc.Definitions["Gizmo"]
	assert.Contains(t, gizmo.Description, "- light",
		"plus bullets must be identified as a list (go-swagger#1726)")
	assert.Contains(t, gizmo.Description, "- sturdy")

	// Wide propagation: the value-list path (Produces/Consumes via AsList) recognises markdown bullets
	// too, not just prose descriptions.
	op := doc.Paths.Paths["/widgets"].Get
	require.NotNil(t, op)
	assert.ElementsMatch(t, []string{"application/json", "application/xml"}, op.Produces,
		"asterisk-bulleted Produces must be parsed as a list (go-swagger#1726)")
	assert.ElementsMatch(t, []string{"application/json"}, op.Consumes,
		"plus-bulleted Consumes must be parsed as a list (go-swagger#1726)")
}
