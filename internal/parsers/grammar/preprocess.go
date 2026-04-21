// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/ast"
	"go/token"
	"strings"
)

// Line is one preprocessed comment line ready for the lexer.
//
// Text has the Go comment markers (// /* */) stripped, along with
// leading continuation decorations common in godoc comments (spaces,
// tabs, asterisks, slashes, dashes, optional markdown table pipe).
// Internal content and embedded whitespace are preserved — fence-body
// indentation handling lives at the lexer layer where fence state is
// tracked.
//
// Pos is the position of Text's first character in the source file:
// Line/Column are accurate on every line (including continuation
// lines inside a /* … */ block) and Offset is the byte offset from
// the start of the file.
type Line struct {
	Text string
	Pos  token.Position
}

// Preprocess turns a comment group into a position-tagged []Line.
//
// Nil CommentGroup or FileSet returns nil. The function is pure: it
// makes no syscalls, allocates a slice proportional to the number of
// physical lines, and is safe for concurrent use.
//
// See architecture §3.1 (stage diagram) and tasks P1.1.
func Preprocess(cg *ast.CommentGroup, fset *token.FileSet) []Line {
	if cg == nil || fset == nil {
		return nil
	}
	var out []Line
	for _, c := range cg.List {
		out = append(out, stripComment(c.Text, fset.Position(c.Slash))...)
	}
	return out
}

// stripComment returns one Line per physical source line of a single
// *ast.Comment. It handles both the `//` line-comment form and the
// `/* … */` block form, including multi-line blocks. Each emitted
// Line's Pos points precisely to the first character of Text in the
// source file (Line, Column, and Offset all accurate).
func stripComment(raw string, basePos token.Position) []Line {
	const markerLen = 2 // "//" and "/*" are both 2 bytes
	switch {
	case strings.HasPrefix(raw, "//"):
		pos := basePos
		pos.Column += markerLen
		pos.Offset += markerLen
		return []Line{stripLine(raw[markerLen:], pos)}

	case strings.HasPrefix(raw, "/*"):
		body := strings.TrimSuffix(raw[markerLen:], "*/")
		out := []Line{}
		lineOffset := 0 // byte index into body where the current line begins
		for lineIdx := 0; ; lineIdx++ {
			nl := strings.IndexByte(body[lineOffset:], '\n')

			var segment string
			if nl < 0 {
				segment = body[lineOffset:]
			} else {
				segment = body[lineOffset : lineOffset+nl]
			}

			pos := basePos
			if lineIdx == 0 {
				// Same source line as basePos; advance past "/*".
				pos.Column += markerLen
				pos.Offset += markerLen
			} else {
				// Continuation line: column restarts at 1; offset is
				// the file offset of the first character of this line.
				pos.Line += lineIdx
				pos.Column = 1
				pos.Offset += markerLen + lineOffset
			}
			out = append(out, stripLine(segment, pos))

			if nl < 0 {
				break
			}
			lineOffset += nl + 1
		}
		return out

	default:
		// Not a valid Go comment; preserve input defensively so
		// downstream layers can surface a diagnostic rather than
		// silently lose data.
		return []Line{{Text: raw, Pos: basePos}}
	}
}

// stripLine trims the leading decoration of a single line and advances
// pos by the number of bytes consumed. pos must already point to the
// first character of the (unstripped) line in the source.
func stripLine(s string, pos token.Position) Line {
	stripped := trimContentPrefix(s)
	consumed := len(s) - len(stripped)
	pos.Column += consumed
	pos.Offset += consumed
	return Line{Text: stripped, Pos: pos}
}

// trimContentPrefix removes the leading godoc-style decoration that
// precedes real content on a comment line:
//   - whitespace (space, tab)
//   - continuation slashes and asterisks (“//“, “ * “, “ *  “)
//   - dashes (“ -- “)
//   - an optional single markdown table pipe “|“
//
// The set mirrors the v1 parser's rxUncommentHeaders so migrated
// fixtures match byte-for-byte at the parse-output level (pre-P5
// parity harness).
func trimContentPrefix(s string) string {
	s = strings.TrimLeft(s, " \t*/-")
	s = strings.TrimPrefix(s, "|")
	return strings.TrimLeft(s, " \t")
}
