// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"bytes"
	"go/scanner"
	gotoken "go/token"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// annotationPrefix is what makes a comment payload rather than prose in this
// app. Comments carrying it are the ONLY reason the source pane exists.
const annotationPrefix = "swagger:"

// goToken is one token of the scan, retained rather than emitted on the spot.
// Comments cannot be classified in a single forward pass: whether a `required:`
// line is grammar or prose depends on whether the FILE carries annotations at
// all, which is only known once the last line has been read.
type goToken struct {
	line, col int // 0-based line, 1-based rune column
	tok       gotoken.Token
	lit       string
}

// BuildGoHighlight classifies Go source into the same per-line lexical runs the
// spec pane uses, so both panes share one renderer and one palette.
//
// The classifier is the standard library's own scanner: it is the definition of
// how Go tokenizes, it costs no dependency, and it is deliberately error
// TOLERANT — a buffer the user is halfway through editing still yields a usable
// token stream instead of nothing. Scan errors are therefore discarded rather
// than reported; a highlighter that gives up on a syntactically incomplete file
// is a highlighter that goes blank exactly when you are typing.
//
// Comments get three classes rather than one, because in a spec generator a
// comment is not uniformly commentary:
//
//   - a `swagger:` line is the annotation that declares the thing, and reads as
//     a spec key — the input that produced the pane next to it;
//   - a leading `<keyword>:` inside an annotated block is grammar, and reads as
//     a keyword, so `// required: true` looks the way `"required": true` does on
//     the spec side;
//   - everything else is prose, and is dimmed.
func BuildGoHighlight(src []byte) *HighlightIndex {
	tokens := scanGo(src)
	annotated := hasAnnotation(tokens)

	byLine := make(map[int][]theme.Span)
	for _, t := range tokens {
		addGoToken(byLine, t, annotated)
	}

	return &HighlightIndex{byLine: byLine}
}

// scanGo tokenizes src into source order.
func scanGo(src []byte) []goToken {
	fset := gotoken.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	starts := lineStarts(src)

	var s scanner.Scanner
	s.Init(file, src, nil, scanner.ScanComments) // nil handler: errors are not our business

	var out []goToken
	for {
		pos, tok, lit := s.Scan()
		if tok == gotoken.EOF {
			break
		}
		// The scanner synthesises the semicolons Go's grammar requires but the
		// source does not write. Marking a run at a character that is not there
		// would style the padding past the end of the line.
		if tok == gotoken.SEMICOLON && lit == "\n" {
			continue
		}

		line := file.Line(pos) - 1
		col := runeColumn(src, starts, line, file.Offset(pos))
		if col < 1 {
			continue
		}

		out = append(out, goToken{line: line, col: col, tok: tok, lit: lit})
	}

	return out
}

// hasAnnotation reports whether the file carries any annotation at all, which
// is what scopes keyword highlighting.
//
// The FILE is the right unit, and the comment group is not. A field's doc
// comment holds the validation keywords while the `swagger:model` that makes
// them meaningful sits on the enclosing TYPE — so scoping per group lights up
// route and parameter bodies but misses every field constraint, which is most
// of them. Scoping per file follows the convention instead: `name`, `in` and
// `example` are ordinary English words, but in a file that already declares
// annotations a comment leading with one is the keyword far more often than not.
//
// What this cannot know is which declarations the scanner actually visits, so a
// keyword-shaped line in an unrelated comment of an annotated file still lights
// up. That needs the AST, and the AST needs a file that parses — which the
// buffer being edited may not.
func hasAnnotation(tokens []goToken) bool {
	for _, t := range tokens {
		if t.tok != gotoken.COMMENT {
			continue
		}
		if slices.ContainsFunc(strings.Split(t.lit, "\n"), isAnnotationComment) {
			return true
		}
	}

	return false
}

// addGoToken records the runs one token contributes, which is more than one when
// the token spans lines. A raw string or a block comment covers every line it
// crosses, and a continuation line carries no token of its own — without a run
// starting at its column 1 it would render plain, and the closing line would go
// plain up to whatever token follows the comment.
func addGoToken(byLine map[int][]theme.Span, t goToken, annotated bool) {
	kind := goSyntaxKind(t.tok, t.lit)

	segments := []string{t.lit}
	if strings.Contains(t.lit, "\n") {
		segments = strings.Split(t.lit, "\n")
	}

	for i, segment := range segments {
		// A continuation line of an empty segment is a BLANK line inside the
		// comment or string. It has no character to mark, and a run there would
		// style the padding the pane fills the row with.
		if i > 0 && segment == "" {
			continue
		}

		line, col := t.line+i, t.col
		if i > 0 {
			col = 1
		}

		if t.tok == gotoken.COMMENT {
			addCommentSpans(byLine, line, col, segment, annotated)

			continue
		}

		byLine[line] = append(byLine[line], theme.Span{Col: col, Kind: kind})
	}
}

// addCommentSpans records the runs on one line of a comment. Classification is
// per LINE, so it catches an annotation trailing a declaration
// (AfterDeclComments) and one buried in a block comment just as well as a
// conventional doc block.
func addCommentSpans(byLine map[int][]theme.Span, line, col int, segment string, annotated bool) {
	add := func(at int, kind theme.SyntaxKind) {
		byLine[line] = append(byLine[line], theme.Span{Col: at, Kind: kind})
	}

	if isAnnotationComment(segment) {
		add(col, theme.SyntaxKey)

		return
	}

	start, end, ok := grammarKeyword(segment)
	if !ok || !annotated {
		add(col, theme.SyntaxComment)

		return
	}

	// The comment markers before the keyword stay prose; only the keyword
	// itself is lifted out, and the value after it returns to prose.
	if start > 0 {
		add(col, theme.SyntaxComment)
	}
	add(col+start, theme.SyntaxKeyword)
	add(col+end, theme.SyntaxComment)
}

// grammarKeyword locates a leading `<keyword>:` in a comment line and returns
// the keyword's rune offsets within it.
//
// Only the text before the FIRST colon is considered, because that is the only
// place the grammar reads a keyword: prose that merely contains a colon does not
// match, and neither does a word the keyword table does not know. The table is
// the parser's own, so what lights up is exactly what the parser will act on —
// aliases (`min` → minimum, `min length` → minLength) and letter case included.
func grammarKeyword(segment string) (start, end int, ok bool) {
	body := strings.TrimLeft(segment, " \t/*")
	lead := len([]rune(segment)) - len([]rune(body))

	name, _, found := strings.Cut(body, ":")
	if !found {
		return 0, 0, false
	}
	if name = strings.TrimSpace(name); name == "" {
		return 0, 0, false
	}
	if _, known := grammar.Lookup(name); !known {
		return 0, 0, false
	}

	return lead, lead + len([]rune(name)), true
}

// goSyntaxKind maps a Go token onto the shared, language-neutral classes.
// Anything unclassified stays SyntaxPlain — identifiers are the bulk of Go
// source, and colouring them would leave nothing uncoloured to contrast against.
func goSyntaxKind(tok gotoken.Token, lit string) theme.SyntaxKind {
	switch {
	case tok == gotoken.COMMENT:
		return theme.SyntaxComment
	case tok == gotoken.STRING, tok == gotoken.CHAR:
		return theme.SyntaxString
	case tok == gotoken.INT, tok == gotoken.FLOAT, tok == gotoken.IMAG:
		return theme.SyntaxNumber
	case tok.IsKeyword(), tok == gotoken.IDENT && isPredeclaredConst(lit):
		return theme.SyntaxKeyword
	case tok.IsOperator():
		return theme.SyntaxPunct
	default:
		return theme.SyntaxPlain
	}
}

// isPredeclaredConst reports whether an identifier is one Go treats as a
// constant rather than a keyword. Colouring them as keywords is what every
// editor does, and `nil` is far too common to read as an ordinary name.
func isPredeclaredConst(lit string) bool {
	switch lit {
	case "nil", "true", "false", "iota":
		return true
	default:
		return false
	}
}

// isAnnotationComment reports whether a comment line leads with a swagger
// annotation, ignoring the comment markers and indentation around it.
func isAnnotationComment(segment string) bool {
	return strings.HasPrefix(strings.TrimLeft(segment, " \t/*"), annotationPrefix)
}

// lineStarts records the byte offset each 0-based line begins at.
func lineStarts(src []byte) []int {
	starts := make([]int, 1, bytes.Count(src, []byte{'\n'})+1)
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}

	return starts
}

// runeColumn converts a byte offset into the 1-based RUNE column the renderer
// slices on. go/token reports columns in bytes, so any multi-byte character
// earlier on the line — an accent in a comment, a symbol in a string — would
// otherwise shift every run after it. Returns 0 for an offset off the line.
func runeColumn(src []byte, starts []int, line, off int) int {
	if line < 0 || line >= len(starts) {
		return 0
	}
	start := starts[line]
	if off < start || off > len(src) {
		return 0
	}

	return utf8.RuneCount(src[start:off]) + 1
}
