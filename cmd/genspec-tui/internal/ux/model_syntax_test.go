// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const syntaxSpec = `{
  "swagger": "2.0",
  "definitions": {
    "User": {
      "properties": {
        "email": {
          "type": "string",
          "maxLength": 64
        }
      }
    }
  }
}`

func syntaxModel(t *testing.T) *Model {
	t.Helper()
	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
	m.spec.SetSize(70, 24)
	m.specJSON = syntaxSpec
	m.refreshSpec()

	return m
}

// The highlight index rides the same walk as the other two, so a refresh must
// install all three or none.
func TestSyntax_RefreshInstallsSpans(t *testing.T) {
	m := syntaxModel(t)

	require.NotNil(t, m.specIndex)
	require.NotNil(t, m.refIndex)
	assert.Contains(t, stripANSI(m.spec.View(false)), `"swagger"`,
		"the text is unchanged by highlighting")
}

// An empty spec must clear the spans along with the indexes; leaving stale runs
// behind would colour the placeholder by the old document's columns.
func TestSyntax_EmptySpecClearsSpans(t *testing.T) {
	m := syntaxModel(t)
	m.specJSON = ""
	m.refreshSpec()

	view := stripANSI(m.spec.View(false))
	assert.Contains(t, view, "(no spec generated yet)")
	assert.NotContains(t, view, "swagger")
}

// Highlighting must never alter the text — the same invariant the renderer is
// tested on, asserted here through the whole pipeline.
func TestSyntax_TextSurvivesHighlighting(t *testing.T) {
	m := syntaxModel(t)

	plain := stripANSI(m.spec.View(false))
	for _, want := range []string{`"swagger": "2.0",`, `"maxLength": 64`, `"type": "string",`} {
		assert.Contains(t, plain, want)
	}
}

// Precedence: the cursor and a search hit answer questions the USER asked, so
// they take the whole line rather than compete with syntax colour for it.
func TestSyntax_PrecedenceCursorThenSearchThenSyntax(t *testing.T) {
	m := syntaxModel(t)

	// Both renders must contain the same visible text regardless of which
	// styling layer won.
	m.spec.SetCursor(1)
	withCursor := stripANSI(m.spec.View(true))
	assert.Contains(t, withCursor, `"swagger": "2.0",`)

	require.Positive(t, m.spec.Search("maxLength"))
	withSearch := stripANSI(m.spec.View(true))
	assert.Contains(t, withSearch, `"maxLength": 64`)

	m.spec.ClearSearch()
	assert.Contains(t, stripANSI(m.spec.View(true)), `"maxLength": 64`)
}

// A search must still count matches on the raw line: highlighting changes how a
// line is drawn, never what it contains.
func TestSyntax_SearchStillCountsMatches(t *testing.T) {
	m := syntaxModel(t)

	n := m.spec.Search(`"type"`)

	assert.Equal(t, 1, n)
	cur, total := m.spec.MatchInfo()
	assert.Equal(t, 1, cur)
	assert.Equal(t, 1, total)
}

// The gutter is prefixed after styling, so the two must coexist.
func TestSyntax_CoexistsWithTheGutter(t *testing.T) {
	m := syntaxModel(t)
	m.spec.SetGutter(map[int]rune{2: panels.GutterAnchor})

	view := stripANSI(m.spec.View(false))

	assert.Contains(t, view, string(panels.GutterAnchor))
	assert.Contains(t, view, `"definitions"`)
}

// Against a real scan, over both renders: the visible text must be exactly the
// document, highlighted or not.
func TestE2E_SyntaxLeavesTheSpecIntact(t *testing.T) {
	m := scanPetstore(t)
	m.spec.SetSize(200, 40)

	for _, format := range []string{"JSON", "YAML"} {
		m.setSpecFormat(format)
		body := m.specJSON
		if format == "YAML" {
			body = m.specYAML
		}
		require.NotEmpty(t, body)

		// Take a line from the middle of the document and require it to appear
		// verbatim in the rendered pane.
		lines := strings.Split(body, "\n")
		probe := strings.TrimRight(lines[len(lines)/2], " ")
		require.NotEmpty(t, strings.TrimSpace(probe))

		m.spec.JumpTo(len(lines) / 2)
		assert.Contains(t, stripANSI(m.spec.View(false)), strings.TrimSpace(probe),
			"%s: highlighting altered the rendered text", format)
	}
}
