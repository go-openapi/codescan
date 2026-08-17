// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package safetext_test

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/safetext"
)

func TestSanitize(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"plain ASCII is untouched", "package main", "package main"},
		{"non-ASCII text is untouched", "caf\u00e9 na\u00efve", "caf\u00e9 na\u00efve"},
		{"empty stays empty", "", ""},

		{"SGR colour", "a\x1b[31mred", "a␛[31mred"},
		{"OSC window title", "\x1b]0;pwned\x07", "␛]0;pwned␇"},
		{"bare ESC", "\x1b", "␛"},
		{"NUL", "a\x00b", "a␀b"},
		{"carriage return overwrite", "real\rfake", "real␍fake"},
		{"DEL", "a\x7fb", "a␡b"},

		// C1 commands the terminal exactly as its ESC-prefixed twin does.
		{"C1 CSI", "a\u009b31mred", "a\ufffd31mred"},
		{"C1 OSC", "a\u009d0;t", "a\ufffd0;t"},
		{"C1 boundaries", "\u0080\u009f", "\ufffd\ufffd"},
		{"just outside C1", "~\u00a0", "~\u00a0"},

		{"line feed is encoded", "a\nb", "a␊b"},
		{"tab is encoded", "a\tb", "a␉b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, safetext.Sanitize(tc.in))
		})
	}
}

func TestSanitizeBlock(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"line feed survives", "line one\nline two", "line one\nline two"},
		{"tab survives", "left\tright", "left\tright"},
		{"carriage return does not", "real\rfake", "real␍fake"},
		{"ESC does not", "a\x1b[2Jb", "a␛[2Jb"},
		{"C1 does not", "a\u009bb", "a\ufffdb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, safetext.SanitizeBlock(tc.in))
		})
	}
}

// TestSanitizeInvalidUTF8 pins the promise that the result is always drawable.
//
// A lone continuation byte is not a rune at all, and writing it to the terminal writes the byte.
func TestSanitizeInvalidUTF8(t *testing.T) {
	t.Parallel()

	got := safetext.Sanitize("a\x9bb")
	assert.True(t, utf8.ValidString(got))
	require.Equal(t, "a\ufffdb", got)
}

// TestSanitizePreservesRuneCount is the property the renderers depend on.
//
// Syntax runs, diagnostic marks and mouse hit-testing all address a line by rune column, and each is computed against
// text that has not been through here. A substitution that changed the count would move every one of them.
func TestSanitizePreservesRuneCount(t *testing.T) {
	t.Parallel()

	for _, in := range []string{
		"plain",
		"\x1b[31mcolour\x1b[0m",
		"tab\there\nand\ra return",
		"caf\u00e9 na\u00efve \u009b\u007f",
		"\x9b\xc3\x28",
	} {
		assert.Lenf(t, []rune(safetext.Sanitize(in)), len([]rune(in)), "input %q", in)
		assert.Lenf(t, []rune(safetext.SanitizeBlock(in)), len([]rune(in)), "input %q", in)
	}
}

// TestSanitizeLeavesNoControl sweeps the whole rune space the terminal can be commanded through.
func TestSanitizeLeavesNoControl(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	for r := rune(0); r <= 0x9F; r++ {
		b.WriteRune(r)
	}
	in := b.String()

	for _, r := range safetext.Sanitize(in) {
		require.Falsef(t, unicode.IsControl(r), "control rune %U survived Sanitize", r)
	}

	for _, r := range safetext.SanitizeBlock(in) {
		if r == '\n' || r == '\t' {
			continue
		}
		require.Falsef(t, unicode.IsControl(r), "control rune %U survived SanitizeBlock", r)
	}
}

// TestSanitizeIsIdempotent matters because the seams overlap: a file name goes through relTo and then through fit.
func TestSanitizeIsIdempotent(t *testing.T) {
	t.Parallel()

	const in = "a\x1b[31mb \u009bc\x7fd\re"

	once := safetext.Sanitize(in)
	require.Equal(t, once, safetext.Sanitize(once))

	onceBlock := safetext.SanitizeBlock(in)
	require.Equal(t, onceBlock, safetext.SanitizeBlock(onceBlock))
}
