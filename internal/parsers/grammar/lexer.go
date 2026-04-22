// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenKind classifies a preprocessed line. The lexer assigns exactly
// one Kind per line (plus a trailing TokenEOF). Parser dispatch (P1.4)
// is driven off Kind without needing to re-inspect Text.
type TokenKind int

const (
	TokenEOF              TokenKind = iota // end of stream
	TokenBlank                             // empty line (after trim)
	TokenText                              // freeform content (title, description)
	TokenAnnotation                        // "swagger:<name> [args...]"
	TokenKeywordValue                      // "<keyword>: <value>"
	TokenKeywordBlockHead                  // "<keyword>:" (value-less; indicates a block follows)
	TokenYAMLFence                         // "---" delimiter
	TokenRawLine                           // verbatim line inside a YAML fence (Text = Line.Raw)
)

// String renders a TokenKind for debugging and diagnostics.
func (k TokenKind) String() string {
	switch k {
	case TokenEOF:
		return "EOF"
	case TokenBlank:
		return "BLANK"
	case TokenText:
		return "TEXT"
	case TokenAnnotation:
		return "ANNOTATION"
	case TokenKeywordValue:
		return "KEYWORD_VALUE"
	case TokenKeywordBlockHead:
		return "KEYWORD_BLOCK_HEAD"
	case TokenYAMLFence:
		return "YAML_FENCE"
	case TokenRawLine:
		return "RAW_LINE"
	default:
		return "?"
	}
}

// Token is one lexed item. Fields are populated per Kind:
//   - TokenAnnotation: Text = annotation name (e.g., "model"), Args = positional args.
//   - TokenKeywordValue / TokenKeywordBlockHead: Text = canonical keyword name,
//     Keyword = table entry, Value = raw value string (empty for BlockHead),
//     ItemsDepth = number of leading "items." prefixes (0 = none),
//     SourceName = the keyword name as it appeared in source (may be
//     an alias like "max" for canonical "maximum").
//   - TokenText: Text = original line content.
//   - TokenBlank / TokenYAMLFence / TokenEOF: Text is empty.
//
// Pos points to the first source character of the meaningful payload
// (the keyword for KEYWORD_*, "swagger:" for ANNOTATION, the fence for
// YAML_FENCE, the start of text for TEXT).
type Token struct {
	Kind       TokenKind
	Pos        token.Position
	Text       string
	Value      string
	Keyword    *Keyword
	ItemsDepth int
	Args       []string
	SourceName string
	// Raw is the source line with only the comment markers (`//` /
	// `/*`) stripped — internal whitespace, indentation, and list
	// markers are preserved. Populated for TokenText and TokenRawLine,
	// empty otherwise. Consumers that need YAML-style indentation or
	// list-marker fidelity (notably the extensions body parser) read
	// Raw; Text is the cleaned form suitable for regex dispatch.
	Raw string
}

// Lex turns a preprocessed line slice into a token stream terminated
// by TokenEOF. The lexer tracks one bit of state — whether the cursor
// is between a pair of `---` fences — so that YAML bodies survive as
// TokenRawLine tokens with their original indentation intact.
func Lex(lines []Line) []Token {
	out := make([]Token, 0, len(lines)+1)
	inFence := false
	for _, line := range lines {
		tok := lexLine(line, inFence)
		out = append(out, tok)
		if tok.Kind == TokenYAMLFence {
			inFence = !inFence
		}
	}
	out = append(out, Token{Kind: TokenEOF})
	return out
}

// lexLine classifies a single preprocessed line. inFence is true when
// the line sits between an opening `---` and its matching closer; in
// that case everything except the closing fence becomes TokenRawLine
// carrying Line.Raw verbatim.
func lexLine(line Line, inFence bool) Token {
	text := strings.TrimRight(line.Text, " \t")

	// Fence detection is always active — a closing `---` is recognised
	// even mid-body.
	if strings.TrimSpace(text) == "---" {
		return Token{Kind: TokenYAMLFence, Pos: line.Pos}
	}
	if inFence {
		return Token{Kind: TokenRawLine, Pos: line.Pos, Text: line.Raw}
	}
	if text == "" {
		return Token{Kind: TokenBlank, Pos: line.Pos}
	}
	if strings.HasPrefix(text, "swagger:") {
		return lexAnnotation(text, line.Pos)
	}
	// swagger:route is the one annotation allowed to follow a leading
	// godoc-style identifier (e.g. `DoFoo swagger:route GET /pets ...`).
	// See architecture §1.1 C2 / v1 rxRoutePrefix.
	if prefixLen, ok := matchGodocRoutePrefix(text); ok {
		pos := line.Pos
		pos.Column += prefixLen
		pos.Offset += prefixLen
		return lexAnnotation(text[prefixLen:], pos)
	}
	if tok, ok := lexKeyword(text, line.Pos); ok {
		return tok
	}
	return Token{Kind: TokenText, Text: text, Raw: line.Raw, Pos: line.Pos}
}

// matchGodocRoutePrefix returns the byte offset of "swagger:route"
// in s if s has the form "<identifier><whitespace>swagger:route<end|whitespace>".
// Returns (0, false) otherwise. Only "route" gets this exception.
func matchGodocRoutePrefix(s string) (int, bool) {
	identEnd := scanIdentifier(s)
	if identEnd == 0 {
		return 0, false
	}
	wsEnd := identEnd
	for wsEnd < len(s) && (s[wsEnd] == ' ' || s[wsEnd] == '\t') {
		wsEnd++
	}
	if wsEnd == identEnd {
		return 0, false
	}
	const prefix = "swagger:" + labelRoute
	if !strings.HasPrefix(s[wsEnd:], prefix) {
		return 0, false
	}
	// Guard against "swagger:routex" — the annotation name must end.
	after := wsEnd + len(prefix)
	if after < len(s) && s[after] != ' ' && s[after] != '\t' {
		return 0, false
	}
	return wsEnd, true
}

// scanIdentifier returns the byte length of a leading godoc identifier
// in s, or 0 if s does not start with one. Matches v1's
// `\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]*` — a Unicode letter followed by
// letters, digits, hyphens, or connector punctuation (underscore).
func scanIdentifier(s string) int {
	if len(s) == 0 {
		return 0
	}
	r, size := utf8.DecodeRuneInString(s)
	if !unicode.IsLetter(r) {
		return 0
	}
	i := size
	for i < len(s) {
		r, size = utf8.DecodeRuneInString(s[i:])
		if !isIdentCont(r) {
			break
		}
		i += size
	}
	return i
}

func isIdentCont(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) ||
		r == '_' || r == '-'
}

// lexAnnotation parses "swagger:<name> [arg1 arg2 ...]". Malformed
// (empty name) falls back to a TEXT token so the parser can emit a
// diagnostic at the analyzer layer.
func lexAnnotation(text string, pos token.Position) Token {
	rest := strings.TrimPrefix(text, "swagger:")
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return Token{Kind: TokenText, Text: text, Pos: pos}
	}
	return Token{
		Kind: TokenAnnotation,
		Pos:  pos,
		Text: fields[0],
		Args: fields[1:],
	}
}

// lexKeyword tries to parse text as a "[items.]*<keyword>: [value]"
// form. Returns (token, true) on a successful match, (zero, false)
// otherwise — in which case the line is emitted as TEXT upstream.
func lexKeyword(text string, pos token.Position) (Token, bool) {
	rest, depth := stripItemsPrefix(text)

	before, after, found := strings.Cut(rest, ":")
	if !found {
		return Token{}, false
	}

	name := strings.TrimSpace(before)
	kw, ok := Lookup(name)
	if !ok {
		return Token{}, false
	}

	// Advance Pos past any items. prefix we stripped so it points to
	// the keyword itself.
	consumed := len(text) - len(rest)
	kwPos := pos
	kwPos.Column += consumed
	kwPos.Offset += consumed

	value := strings.TrimSpace(after)

	kind := TokenKeywordValue
	if value == "" {
		kind = TokenKeywordBlockHead
	}

	return Token{
		Kind:       kind,
		Pos:        kwPos,
		Text:       kw.Name,
		Value:      value,
		Keyword:    &kw,
		ItemsDepth: depth,
		SourceName: name,
	}, true
}

// stripItemsPrefix removes leading `items.`, `items `, or `items\t`
// runs from s, counting how many were stripped. The v1 form is
// captured by rxItemsPrefixFmt = `(?:[Ii]tems[\.\p{Zs}]*){%d}`;
// this is the equivalent hand-rolled recognizer.
//
// Notably it does *not* strip `items` with no following separator
// (so "items:" as a standalone keyword remains intact), and it does
// not match inside longer identifiers ("maxItems" stays as a single
// word because "items" doesn't appear at the *start*).
func stripItemsPrefix(s string) (rest string, depth int) {
	for {
		stripped, ok := stripOneItemsPrefix(s)
		if !ok {
			return s, depth
		}
		s = stripped
		depth++
	}
}

func stripOneItemsPrefix(s string) (string, bool) {
	const itemsLen = 5 // len("items")
	if len(s) < itemsLen {
		return s, false
	}
	if !strings.EqualFold(s[:itemsLen], "items") {
		return s, false
	}
	rest := s[itemsLen:]
	trimmed := strings.TrimLeft(rest, ". \t")
	if len(trimmed) == len(rest) {
		// No separator — "items" is part of a longer identifier
		// (e.g., "itemspan") or the bare keyword "items:".
		return s, false
	}
	return trimmed, true
}
