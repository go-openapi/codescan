// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const referenceGo = `package models

// User is a user.
//
// swagger:model User
type User struct {
	// swagger:name emailAddress
	Email string
}

// plain comment, nothing to look up
var x = "swagger:model in a string literal"
`

// pressK sends the reference key the way a terminal sends a capital.
func pressK(m *Model) tea.Cmd {
	return m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
}

// refModel opens the fixture in the viewer with the nav line on the given 0-based row.
func refModel(t *testing.T, line int) *Model {
	t.Helper()

	m := testModel(t, sized(120, 40), viewing("models.go", referenceGo), focusedOn(paneTree))
	m.fileView.GotoLine(line)

	return m
}

func TestReference_ShowsTheAnnotationUnderTheCursor(t *testing.T) {
	m := refModel(t, 4) // "// swagger:model User"

	require.Nil(t, pressK(m))

	require.True(t, m.reference.IsOpen())
	view := testutils.StripANSI(m.reference.View())
	assert.Contains(t, view, "swagger:model")
	assert.Contains(t, view, "definitions entry", "the summary is shown")
	assert.Contains(t, view, "required", "and what may go in its body")
}

// The popup is per-line, so moving the cursor changes what it answers about.
func TestReference_FollowsTheCursor(t *testing.T) {
	m := refModel(t, 6) // "// swagger:name emailAddress"

	_ = pressK(m)

	assert.Contains(t, testutils.StripANSI(m.reference.View()), "swagger:name")
}

// TestReference_IgnoresAStringLiteral pins the scope of the lookup.
//
// The scanner's own fixtures and tests are full of annotation text inside string literals; offering one as a live
// annotation would claim the scanner reads it, which it does not.
func TestReference_IgnoresAStringLiteral(t *testing.T) {
	m := refModel(t, 11) // `var x = "swagger:model in a string literal"`

	_ = pressK(m)

	assert.False(t, m.reference.IsOpen(), "text inside a string literal is not an annotation")
}

func TestReference_NoAnnotationOnTheLine(t *testing.T) {
	m := refModel(t, 0) // "package models"

	_ = pressK(m)

	assert.False(t, m.reference.IsOpen())
	assert.Contains(t, m.notice, "no swagger annotation")
}

// TestReference_KDoesNotMoveTheCursor is the collision guard: bindings are matched case-insensitively, so `K` would
// otherwise be read as `k` and scroll the viewer.
func TestReference_KDoesNotMoveTheCursor(t *testing.T) {
	m := refModel(t, 4)

	_ = pressK(m)

	assert.Equal(t, 4, m.fileView.CurrentLine(), "K is a lookup, not a movement")
}

// TestReference_ReadsTheBuffer pins that the lookup works on an annotation still being typed, which is when it is most
// wanted — the file on disk does not have it yet.
func TestReference_ReadsTheBuffer(t *testing.T) {
	m := testModel(t, sized(120, 40), viewing("models.go", "package p\n// x\n"), focusedOn(paneTree))
	m.fileView.StartEdit()
	m.fileView.GotoLine(1)
	_ = m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("// swagger:enum Kind")})
	m.fileView.StopEdit()

	_ = pressK(m)

	require.True(t, m.reference.IsOpen())
	assert.Contains(t, testutils.StripANSI(m.reference.View()), "swagger:enum")
}

// TestReference_DismissKeys pins that the popup goes away, including via the key that opened it.
func TestReference_DismissKeys(t *testing.T) {
	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyEsc},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'K'}},
	} {
		m := refModel(t, 4)
		_ = pressK(m)
		require.True(t, m.reference.IsOpen())

		_ = m.handleKey(msg)

		assert.False(t, m.reference.IsOpen(), "%s must dismiss the reference", msg.String())
	}
}

// TestReference_EveryDocumentedAnnotationIsReachable ties the TUI's lookup to the grammar's table: every annotation the
// grammar documents must be findable from the text a user actually types.
func TestReference_EveryDocumentedAnnotationIsReachable(t *testing.T) {
	for a := grammar.AnnModel; a <= grammar.AnnDescription; a++ {
		line := "// " + grammar.AnnotationPrefix + a.String() + " something"

		site, ok := annotationOnLine(line)

		require.True(t, ok, "%q is not recognised as an annotation", line)
		assert.Equal(t, a, site.Kind)

		// The token range must cover exactly `swagger:<name>` — the click target.
		assert.Equal(t, 4, site.Start, "the directive starts after `// `")
		assert.Equal(t, 4+len(grammar.AnnotationPrefix+a.String()), site.End)
		assert.True(t, site.covers(site.Start) && site.covers(site.End-1))
		assert.False(t, site.covers(site.Start-1), "the space before the directive is not part of it")
		assert.False(t, site.covers(site.End), "nor is the argument after it")
	}
}

// TestAnnotationSite_ColumnsAreRunes pins the token range against a comment carrying multi-byte text before the
// directive, where a byte offset would place the click target several columns to the right of what is drawn.
func TestAnnotationSite_ColumnsAreRunes(t *testing.T) {
	site, ok := annotationOnLine("// é—çà " + grammar.AnnotationPrefix + "model User")

	require.True(t, ok)
	assert.Equal(t, 9, site.Start, "five runes of prose after `// `, not their byte length")
	assert.Equal(t, 9+len("swagger:model"), site.End)
}

// TestAnnotationSite_RejectsASpaceAfterThePrefix pins that the name must be adjacent, as the grammar requires — and
// that a near-miss does not produce a token range with a hole in it.
func TestAnnotationSite_RejectsASpaceAfterThePrefix(t *testing.T) {
	_, ok := annotationOnLine("// " + grammar.AnnotationPrefix + " model")

	assert.False(t, ok)
}

// clickAt sends a left-press at absolute terminal coordinates.
func clickAt(m *Model, x, y int) tea.Cmd {
	return m.handleMouse(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
}

// annotationPoint locates the on-screen position of a run of text on a given buffer line, so a click can be aimed at
// what is DRAWN rather than at coordinates guessed from the source.
func annotationPoint(t *testing.T, m *Model, line int, needle string) (x, y int) {
	t.Helper()

	text, ok := m.fileView.Line(line)
	require.True(t, ok)
	idx := strings.Index(text, needle)
	require.GreaterOrEqual(t, idx, 0, "%q is not on line %d", needle, line)

	// Invert LineColAt: find the point that maps back to this line and column.
	for y := range 40 {
		for x := range 120 {
			gotLine, gotCol, ok := m.fileView.LineColAt(x, y)
			if ok && gotLine == line && gotCol == idx+1 {
				return x, y + headerH // panel-relative → absolute
			}
		}
	}
	t.Fatalf("no screen point maps to line %d col %d", line, idx+1)

	return 0, 0
}

func TestReference_ClickOnAnAnnotationOpensIt(t *testing.T) {
	m := refModel(t, 0)
	x, y := annotationPoint(t, m, 4, "swagger:model")

	_ = clickAt(m, x, y)

	require.True(t, m.reference.IsOpen())
	assert.Contains(t, testutils.StripANSI(m.reference.View()), "swagger:model")
}

// TestReference_ClickElsewhereJustFocuses is the guard against the popup becoming a nuisance: clicking a pane to focus
// it is the most ordinary thing a user does, and it must not throw a modal up.
func TestReference_ClickElsewhereJustFocuses(t *testing.T) {
	m := refModel(t, 0)
	m.focused = paneSpec

	// The argument AFTER the directive, on the very same line.
	x, y := annotationPoint(t, m, 4, "User")
	_ = clickAt(m, x, y)

	assert.False(t, m.reference.IsOpen(), "only the directive token itself opens the reference")
	assert.Equal(t, paneTree, m.focused, "the click still focused the pane")
}

// TestReference_ClickDoesNotMoveTheNavLine pins that a click asks "what is that", not "go there".
func TestReference_ClickDoesNotMoveTheNavLine(t *testing.T) {
	m := refModel(t, 0)
	x, y := annotationPoint(t, m, 4, "swagger:model")

	_ = clickAt(m, x, y)

	assert.Equal(t, 0, m.fileView.CurrentLine())
}

// TestFileView_LineColAtAgreesWithWhatIsDrawn is the geometry pin for click targeting.
//
// LineColAt reproduces a layout computed by the renderer. If the two drift by a column, a click near the edge of a
// token resolves to the wrong thing — and it stays plausible, because the middle of every token still works. So this
// checks the mapping against the actual frame rather than against the arithmetic that produced it.
func TestFileView_LineColAtAgreesWithWhatIsDrawn(t *testing.T) {
	m := refModel(t, 0)
	frame := strings.Split(testutils.StripANSI(m.fileView.View(true, false)), "\n")
	checked := 0

	for y, row := range frame {
		for x, drawn := range []rune(row) {
			line, col, ok := m.fileView.LineColAt(x, y)
			if !ok {
				continue
			}

			text, found := m.fileView.Line(line)
			require.True(t, found, "mapping produced line %d, which does not exist", line)
			runes := []rune(text)
			if col-1 >= len(runes) {
				continue // past the end of that line's text: the renderer pads with blanks
			}

			assert.Equal(t, string(runes[col-1]), string(drawn),
				"at screen (%d,%d) the mapping says line %d col %d", x, y, line, col)
			checked++
		}
	}

	assert.Positive(t, checked, "the frame produced no mappable cells; the test proved nothing")
}
