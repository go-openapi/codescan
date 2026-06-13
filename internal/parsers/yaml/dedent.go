// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package yaml

import "strings"

// RemoveIndent normalises the common leading indentation of a YAML
// body lifted from a godoc comment block: the first non-blank line's
// indent length is treated as the canonical strip width and applied to
// every line. Any tabs in the stripped lines' leading-whitespace run
// are then expanded to two spaces, because YAML refuses tab
// indentation.
//
// The first-(non-blank-)line dedent (vs "shortest leading-whitespace
// run across every non-blank line") is the operations / meta path's
// contract — the existing operation goldens depend on it. The
// typed-extensions path uses common-prefix dedent instead; see
// README.md §dedent.
//
// Leading blank lines are skipped when choosing the canonical line:
// gofmt inserts a blank "//" line directly under a column-0
// doc-comment key (its code-block rule), so the gofmt-canonical
// swagger:meta form is `key:` at column 0, a blank line, then
// tab-indented children. Keying the strip width off the literal first
// line would see the blank line, derive a zero strip width, and leave
// the children's tabs intact for the YAML parser to reject ("found
// character that cannot start any token"). Skipping the blank line
// makes that form dedent and retab identically to the all-indented
// variant (quirk F7).
//
// Whitespace tokens recognised here are space (' '), tab ('\t'), and
// the leading `/` characters that survive when the lexer hasn't
// stripped a godoc comment marker yet. Unicode space separators
// (\p{Zs}) are NOT recognised: real Go source code uses ASCII
// whitespace. If a corpus surfaces that depends on it, reintroduce
// the Unicode branch.
func RemoveIndent(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}

	first := 0
	for first < len(lines) && strings.TrimSpace(lines[first]) == "" {
		first++
	}
	if first == len(lines) {
		return lines
	}

	indent := leadingIndent(lines[first])
	if indent == 0 {
		return lines
	}

	out := make([]string, len(lines))
	for i, line := range lines {
		if len(line) < indent {
			out[i] = line
			continue
		}
		out[i] = retabLeading(line[indent:])
	}
	return out
}

// leadingIndent returns the byte position of the first non-indent
// character on line — i.e., the number of bytes the line should be
// stripped by to remove its leading indent.
//
// An "indent" here is the maximal prefix matching the pattern
// (ws* / ws*)+ followed by ws*, where ws is space or tab. The
// optional `/` run absorbs godoc comment markers (`//`, `///`) when
// they slipped through preprocessing.
//
// Lines that are pure indent (empty or all-whitespace) return 0 —
// there's no meaningful strip count for them.
func leadingIndent(line string) int {
	i := 0
	for i < len(line) && isIndentSpace(line[i]) {
		i++
	}
	for i < len(line) && line[i] == '/' {
		i++
	}
	for i < len(line) && isIndentSpace(line[i]) {
		i++
	}
	if i >= len(line) {
		return 0
	}
	return i
}

// retabLeading replaces every tab in the leading-whitespace run of
// line with two spaces. Tabs past the first non-whitespace character
// are left untouched.
func retabLeading(line string) string {
	n := 0
	for n < len(line) && isIndentSpace(line[n]) {
		n++
	}
	if n == 0 {
		return line
	}
	return strings.ReplaceAll(line[:n], "\t", "  ") + line[n:]
}

func isIndentSpace(b byte) bool { return b == ' ' || b == '\t' }
