// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/token"
	"strings"
)

// TokenKind classifies a preprocessed line. The lexer assigns exactly
// one Kind per line (plus a trailing TokenEOF). Parser dispatch (P1.4)
// is driven off Kind without needing to re-inspect Text.
type TokenKind int

const (
	TokenEOF              TokenKind = iota // end of stream
	TokenBlank                             // empty line (after trim)
	TokenText                              // freeform content (title, description, block body)
	TokenAnnotation                        // "swagger:<name> [args...]"
	TokenKeywordValue                      // "<keyword>: <value>"
	TokenKeywordBlockHead                  // "<keyword>:" (value-less; indicates a block follows)
	TokenYAMLFence                         // "---" delimiter
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
	default:
		return "?"
	}
}

// Token is one lexed item. Fields are populated per Kind:
//   - TokenAnnotation: Text = annotation name (e.g., "model"), Args = positional args.
//   - TokenKeywordValue / TokenKeywordBlockHead: Text = canonical keyword name,
//     Keyword = table entry, Value = raw value string (empty for BlockHead),
//     ItemsDepth = number of leading "items." prefixes (0 = none).
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
}

// Lex turns a preprocessed line slice into a token stream terminated
// by TokenEOF. The lexer is context-free (no fence/state tracking);
// the parser decides whether a TokenText sits inside a YAML body.
func Lex(lines []Line) []Token {
	out := make([]Token, 0, len(lines)+1)
	for _, line := range lines {
		out = append(out, lexLine(line))
	}
	out = append(out, Token{Kind: TokenEOF})
	return out
}

// lexLine classifies a single preprocessed line.
func lexLine(line Line) Token {
	text := strings.TrimRight(line.Text, " \t")

	if text == "" {
		return Token{Kind: TokenBlank, Pos: line.Pos}
	}
	if strings.TrimSpace(text) == "---" {
		return Token{Kind: TokenYAMLFence, Pos: line.Pos}
	}
	if strings.HasPrefix(text, "swagger:") {
		return lexAnnotation(text, line.Pos)
	}
	if tok, ok := lexKeyword(text, line.Pos); ok {
		return tok
	}
	return Token{Kind: TokenText, Text: text, Pos: line.Pos}
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
