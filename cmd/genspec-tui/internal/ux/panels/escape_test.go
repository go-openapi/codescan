// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// The payloads every assertion below is written against.
//
// Each carries a marker no style this package emits would ever produce after an ESC, so "the raw form is absent and
// the encoded form is present" is a statement about the scanned text and not about the panel's own colours - which
// the package's TestMain deliberately turns on.
const (
	rawSGR     = "\x1b[31mPWNED"
	encodedSGR = "␛[31mPWNED"

	rawOSC     = "\x1b]0;PWNED\x07"
	encodedOSC = "␛]0;PWNED␇"
)

// assertEncoded is the shared check: the terminal is handed the picture, never the command.
func assertEncoded(t *testing.T, out, raw, encoded string) {
	t.Helper()

	assert.NotContains(t, out, raw, "a control sequence from the scanned tree reached the terminal")
	assert.Contains(t, out, encoded, "the control sequence was dropped instead of being shown")
}

func TestFit_EncodesControlSequences(t *testing.T) {
	got := fit(rawSGR, 40)

	assertEncoded(t, got, rawSGR, encodedSGR)
	assert.Len(t, []rune(got), 40, "padding still counts the encoded runes")
}

// TestFit_TruncatesEncoded pins that the encoding happens BEFORE the width is applied.
//
// Sanitizing after the cut would let a sequence be sliced in half, which is the same defect as colouring before
// truncating: what reaches the terminal is then neither the text nor a complete escape.
func TestFit_TruncatesEncoded(t *testing.T) {
	got := fit(rawSGR, 6)

	assert.NotContains(t, got, "\x1b")
	assert.Equal(t, "␛[31m…", got)
}

func TestRenderSpans_EncodesControlSequences(t *testing.T) {
	spans := []theme.Span{{Col: 1, Kind: theme.SyntaxComment}}

	t.Run("with spans", func(t *testing.T) {
		assertEncoded(t, renderSpans(rawSGR, spans, 40), rawSGR, encodedSGR)
	})

	t.Run("without spans", func(t *testing.T) {
		assertEncoded(t, renderSpans(rawOSC, nil, 40), rawOSC, encodedOSC)
	})
}

// TestRenderSpans_SpanColumnsSurvive is the reason the sanitizer replaces one rune with one rune.
//
// The runs are computed against the file's own text, so an encoding that changed a line's length would colour the
// wrong columns of it.
func TestRenderSpans_SpanColumnsSurvive(t *testing.T) {
	// "ab" + ESC + "cd": the second run starts on the "c", at rune column 4.
	const raw = "ab\x1bcd"

	spans := []theme.Span{
		{Col: 1, Kind: theme.SyntaxComment},
		{Col: 4, Kind: theme.SyntaxKeyword},
	}
	got := renderSpans(raw, spans, 5)

	assert.Contains(t, got, theme.Syntax(theme.SyntaxComment).Render("ab␛"))
	assert.Contains(t, got, theme.Syntax(theme.SyntaxKeyword).Render("cd"))
}

// TestFileView_TitleEncodesControlSequences covers the one row of the panel that does not go through fit.
func TestFileView_TitleEncodesControlSequences(t *testing.T) {
	fv := NewFileView()
	fv.SetSize(60, 10)
	fv.SetFile("pkg/"+rawSGR+".go", "package p\n")

	assertEncoded(t, fv.View(true, true), rawSGR, encodedSGR)
}

// TestFileView_BodyEncodesControlSequences is belt and braces.
//
// bubbles/textarea drops every control rune when the buffer is set, so a crafted file is already declawed by the time
// the viewer splits it into rows. That is the widget's paste hygiene rather than a promise to us, and it would go away
// with any change of editor - so the renderer guards the same ground.
func TestFileView_BodyEncodesControlSequences(t *testing.T) {
	fv := NewFileView()
	fv.SetSize(60, 10)
	fv.SetFile("x.go", "package p\n// "+rawSGR+"\n")

	assert.NotContains(t, fv.View(false, false), rawSGR)
}

// TestTree_EncodesControlSequences covers the file names the browser lists.
//
// A repository chooses them, and on the platforms this runs on a name may hold anything but a separator and a NUL.
func TestTree_EncodesControlSequences(t *testing.T) {
	dir := t.TempDir()
	name := "evil" + rawSGR + ".go"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("package p\n"), 0o600); err != nil {
		t.Skipf("this filesystem will not hold a name with a control character: %v", err)
	}

	tr := NewTree(dir)
	tr.SetSize(60, 20)
	out := tr.View(true)

	require.Contains(t, out, "evil", "the crafted file is not in the tree at all, so nothing is being tested")
	assertEncoded(t, out, rawSGR, encodedSGR)
}
