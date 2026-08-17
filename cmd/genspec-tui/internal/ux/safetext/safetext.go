// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package safetext

import (
	"strings"
	"unicode/utf8"
)

// The stand-ins a control character is shown as.
//
// The C0 range has a picture of its own for every member: U+2400 stands for NUL and the block runs in step with the
// range up to U+241F for US, so the substitution is one addition. Showing WHAT was there rather than merely that
// something was beats a uniform marker - an ESC drawn as U+241B explains an odd-looking line, and a NUL explains a
// different one.
//
// Nothing else has a picture. DEL has its own at U+2421; the C1 range and bytes that are not valid UTF-8 at all fall
// back to the replacement character, which a reader already understands as "a character that could not be shown".
const (
	c0Picture   = '\u2400'
	del         = '\u007f'
	delPicture  = '\u2421'
	replacement = '\ufffd'
)

// The C1 range: the control characters above ASCII.
//
// They matter because a terminal reading UTF-8 acts on them as it acts on their ESC-prefixed twins - U+009B commands
// as CSI does, U+009D as OSC does - so encoding ESC alone would leave the same attack spelled one rune shorter.
const (
	c1Low  = '\u0080'
	c1High = '\u009f'
)

// Sanitize returns s with every control character replaced by a visible stand-in.
//
// This is the default, and what anything drawn on a single row wants: a file name, a JSON pointer, a status notice,
// one line of a source file. Line feeds and tabs are encoded along with the rest, since a row that grew a line of its
// own would push the layout below it out of step with where the mouse thinks it is.
//
// The result is valid UTF-8 and holds exactly as many runes as s, counting each byte of an invalid sequence as one.
// Text that carries no control character is returned unchanged.
//
// Use [SanitizeBlock] for prose that is allowed to span rows.
func Sanitize(s string) string {
	return sanitize(s, false)
}

// SanitizeBlock returns s with every control character except the line feed and the tab replaced by a visible
// stand-in.
//
// For prose whose own line breaks the pane lays out - a diagnostic message, a validation finding - where encoding the
// breaks would run the paragraph together. Neither survivor can command the terminal; they only ever cost alignment,
// which is the pane's business rather than this package's.
//
// Carriage return is NOT among them. It returns the cursor to the start of the row, which is enough to overwrite what
// has already been drawn and show the reader something other than what was written.
//
// Everything [Sanitize] guarantees about its result holds here too.
func SanitizeBlock(s string) string {
	return sanitize(s, true)
}

// sanitize is the shared pass, keepLayout deciding whether the line feed and the tab are let through.
func sanitize(s string, keepLayout bool) string {
	if !hasControl(s, keepLayout) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(visible(r, keepLayout))
	}

	return b.String()
}

// visible maps one rune onto what is safe to draw for it, and is the identity on everything else.
func visible(r rune, keepLayout bool) rune {
	if keepLayout && (r == '\n' || r == '\t') {
		return r
	}

	switch {
	case r < ' ':
		return c0Picture + r
	case r == utf8.RuneError:
		// Reached for a genuine U+FFFD as well as for a byte that decoded to one.
		// The two cannot be told apart here, and the mapping is the identity either way.
		return replacement
	case r == del:
		return delPicture
	case r >= c1Low && r <= c1High:
		return replacement
	default:
		return r
	}
}

// hasControl reports whether s holds anything [sanitize] would rewrite.
//
// The scan is over bytes rather than runes, so the common case - a row of plain ASCII - costs one pass and no
// decoding. Any byte at or above utf8.RuneSelf sends the string down the slow path, which is where the C1 range and
// the invalid sequences are told apart from ordinary text.
func hasControl(s string, keepLayout bool) bool {
	for i := range len(s) {
		c := s[i]
		if keepLayout && (c == '\n' || c == '\t') {
			continue
		}
		if c < ' ' || c == del || c >= utf8.RuneSelf {
			return true
		}
	}

	return false
}
