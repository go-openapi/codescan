// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package grammar implements the v2 annotation parser.
//
// It replaces the regexp-based parser at internal/parsers/*.go with a
// hand-rolled recursive-descent parser producing a typed Block family
// (see ast.go).
//
// See .claude/plans/grammar-parser-architecture.md for the "why" and
// .claude/plans/grammar-parser-tasks.md for the "how".
package grammar

import (
	"go/ast"
	"go/token"
	"strings"
)

// Parse runs the full preprocess → lex → parse pipeline on a comment
// group and returns the typed Block that describes it. Never panics;
// diagnostics accumulate on the returned Block.
//
// A nil CommentGroup produces an empty UnboundBlock — useful for code
// paths that call Parse unconditionally.
//
//nolint:ireturn // Block is a polymorphic family (ModelBlock, RouteBlock, …); concrete type depends on the annotation.
func Parse(cg *ast.CommentGroup, fset *token.FileSet) Block {
	lines := Preprocess(cg, fset)
	tokens := Lex(lines)
	return ParseTokens(tokens)
}

// ParseTokens runs parser-only on a pre-lexed token stream. Useful
// for tests and for LSP scenarios where the token stream is
// synthesized (e.g., insertion-point completion).
//
//nolint:ireturn // see Parse godoc
func ParseTokens(tokens []Token) Block {
	p := &parseState{tokens: tokens}
	return p.parse()
}

type parseState struct {
	tokens []Token
	diag   []Diagnostic
}

//nolint:ireturn // see Parse godoc
func (p *parseState) parse() Block {
	annIdx := findAnnotation(p.tokens)

	var (
		kind  AnnotationKind
		typed Block
		base  *baseBlock
		pre   []Token
		post  []Token
	)

	if annIdx >= 0 {
		annTok := p.tokens[annIdx]
		kind = AnnotationKindFromName(annTok.Text)
		base = newBaseBlock(kind, annTok.Pos)
		typed = p.buildTypedBlock(kind, annTok, base)
		pre = p.tokens[:annIdx]
		post = p.tokens[annIdx+1:]
	} else {
		base = newBaseBlock(AnnUnknown, firstMeaningfulPos(p.tokens))
		typed = &UnboundBlock{baseBlock: base}
		post = p.tokens
	}

	p.parseTitleDesc(base, pre)
	p.parseBody(base, post)

	base.diagnostics = append(base.diagnostics, p.diag...)
	return typed
}

// findAnnotation returns the index of the first TokenAnnotation in
// tokens, or -1 if none. Annotations normally occupy the top of a
// comment group, but godoc-style placement (e.g., annotation after a
// description paragraph) is accepted and triggers the same dispatch.
func findAnnotation(tokens []Token) int {
	for i, t := range tokens {
		if t.Kind == TokenAnnotation {
			return i
		}
	}
	return -1
}

// firstMeaningfulPos returns the Pos of the first non-blank, non-EOF
// token — i.e., the reasonable "position" of a comment group that has
// no annotation.
func firstMeaningfulPos(tokens []Token) token.Position {
	for _, t := range tokens {
		if t.Kind != TokenBlank && t.Kind != TokenEOF {
			return t.Pos
		}
	}
	return token.Position{}
}

// buildTypedBlock constructs the typed Block that corresponds to the
// recognized annotation kind, populating kind-specific positional
// fields from the annotation's Args.
//
// Unrecognized or v1-parity-simple annotations (strfmt, alias, name,
// allOf, enum, ignore, default, type, file) return an UnboundBlock
// carrying the AnnotationKind — analyzers type-switch on the kind to
// decide further handling.
//
//nolint:ireturn // see Parse godoc
func (p *parseState) buildTypedBlock(kind AnnotationKind, tok Token, base *baseBlock) Block {
	switch kind {
	case AnnModel:
		return &ModelBlock{baseBlock: base, Name: firstArg(tok.Args)}

	case AnnResponse:
		return &ResponseBlock{baseBlock: base, Name: firstArg(tok.Args)}

	case AnnParameters:
		return &ParametersBlock{
			baseBlock:   base,
			TargetTypes: append([]string(nil), tok.Args...),
		}

	case AnnMeta:
		return &MetaBlock{baseBlock: base}

	case AnnRoute:
		rb := &RouteBlock{baseBlock: base}
		p.fillOperationArgs(&rb.Method, &rb.Path, &rb.Tags, &rb.OpID, tok)
		return rb

	case AnnOperation:
		ob := &OperationBlock{baseBlock: base}
		p.fillOperationArgs(&ob.Method, &ob.Path, &ob.Tags, &ob.OpID, tok)
		return ob

	case AnnUnknown,
		AnnStrfmt, AnnAlias, AnnName, AnnAllOf, AnnEnumDecl,
		AnnIgnore, AnnDefaultName, AnnType, AnnFile:
		return &UnboundBlock{baseBlock: base}

	default:
		return &UnboundBlock{baseBlock: base}
	}
}

// fillOperationArgs extracts METHOD, /path, optional tags (free-text
// segment), and opID from the positional args of swagger:route /
// swagger:operation. Matches the v1 regex-based extraction.
func (p *parseState) fillOperationArgs(method, path, tags, opID *string, tok Token) {
	args := tok.Args
	switch {
	case len(args) < minOpArgs:
		p.diag = append(p.diag, Errorf(tok.Pos, CodeInvalidAnnotation,
			"swagger:%s requires method, path, and operation id (got %d args)",
			tok.Text, len(args)))
	case len(args) == minOpArgs:
		*method, *path, *opID = args[0], args[1], args[2]
	default:
		*method, *path = args[0], args[1]
		*tags = strings.Join(args[2:len(args)-1], " ")
		*opID = args[len(args)-1]
	}
}

const minOpArgs = 3 // method + path + opID

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// parseTitleDesc extracts the title (first paragraph) and description
// (remaining paragraphs, joined by blank lines) from the tokens that
// appear before the annotation.
//
// Keyword/YAML/block-head tokens appearing pre-annotation are unusual
// but not fatal — they are ignored with no diagnostic for v1 parity.
func (p *parseState) parseTitleDesc(base *baseBlock, pre []Token) {
	var paragraphs []string
	var current []string

	flush := func() {
		if len(current) > 0 {
			paragraphs = append(paragraphs, strings.Join(current, " "))
			current = current[:0]
		}
	}

	for _, t := range pre {
		switch t.Kind {
		case TokenBlank:
			flush()
		case TokenText:
			current = append(current, t.Text)
		case TokenEOF,
			TokenAnnotation,
			TokenKeywordValue, TokenKeywordBlockHead,
			TokenYAMLFence:
			// Ignored in the title/description slice.
		default:
			// Unreachable at v1; future kinds ignored defensively.
		}
	}
	flush()

	if len(paragraphs) > 0 {
		base.title = paragraphs[0]
	}
	if len(paragraphs) > 1 {
		base.description = strings.Join(paragraphs[1:], "\n\n")
	}
}

// parseBody handles post-annotation tokens: properties
// (KEYWORD_VALUE / KEYWORD_BLOCK_HEAD), YAML fenced bodies, and the
// terminal TokenEOF. Error recovery: skip unknown tokens; never abort.
//
// Scope for P1.4:
//   - KEYWORD_VALUE / KEYWORD_BLOCK_HEAD → Property
//   - YAML_FENCE → capture body between matching fences into RawYAML
//   - TEXT / BLANK → dropped silently (multi-line block-body collection
//     for consumes/produces/security/etc. is P2.3 scope; preserving
//     indentation inside YAML fences is also P2.1 scope)
//   - Stray ANNOTATION → non-fatal diagnostic (one annotation per block)
func (p *parseState) parseBody(base *baseBlock, post []Token) {
	i := 0
	for i < len(post) {
		t := post[i]
		switch t.Kind {
		case TokenEOF, TokenBlank, TokenText:
			i++

		case TokenKeywordValue:
			base.properties = append(base.properties, Property{
				Keyword:    *t.Keyword,
				Pos:        t.Pos,
				Value:      t.Value,
				ItemsDepth: t.ItemsDepth,
			})
			i++

		case TokenKeywordBlockHead:
			base.properties = append(base.properties, Property{
				Keyword:    *t.Keyword,
				Pos:        t.Pos,
				ItemsDepth: t.ItemsDepth,
			})
			i++

		case TokenYAMLFence:
			i = p.collectYAMLBody(base, post, i)

		case TokenAnnotation:
			p.diag = append(p.diag, Warnf(t.Pos, CodeInvalidAnnotation,
				"additional swagger:%s annotation ignored (one per comment block)",
				t.Text))
			i++

		default:
			i++
		}
	}
}

// collectYAMLBody captures everything between a YAML_FENCE opener at
// index i and its matching closer (or EOF). Emits an UnterminatedYAML
// diagnostic if no closer is found. Returns the index past the closer.
//
// NOTE: line reconstruction is best-effort at P1.4 — tokens have been
// classified (so `responses:` became a KEYWORD_BLOCK_HEAD), and
// indentation is lost because the preprocessor already trimmed it.
// P2.1 will add fence-state tracking so raw YAML bytes survive
// verbatim. Until then, YAML bodies captured here are suitable for
// Kind/content detection but not for full YAML re-parsing.
func (p *parseState) collectYAMLBody(base *baseBlock, post []Token, i int) int {
	openerPos := post[i].Pos
	i++

	var body []string
	for i < len(post) && post[i].Kind != TokenYAMLFence && post[i].Kind != TokenEOF {
		body = append(body, reconstructLine(post[i]))
		i++
	}

	if i < len(post) && post[i].Kind == TokenYAMLFence {
		i++ // consume closer
	} else {
		p.diag = append(p.diag, Errorf(openerPos, CodeUnterminatedYAML,
			"YAML body opened with --- but never closed"))
	}

	base.yamlBlocks = append(base.yamlBlocks, RawYAML{
		Pos:  openerPos,
		Text: strings.Join(body, "\n"),
	})
	return i
}

// reconstructLine returns a best-effort text rendering of a Token as
// it appeared on the source line. Used inside YAML fence capture where
// the classifier has already split `keyword:` into KEYWORD_BLOCK_HEAD.
// Indentation is NOT preserved (preprocessor already stripped it);
// P2.1 will fix this.
func reconstructLine(t Token) string {
	switch t.Kind {
	case TokenBlank:
		return ""
	case TokenKeywordValue:
		return t.Text + ": " + t.Value
	case TokenKeywordBlockHead:
		return t.Text + ":"
	case TokenAnnotation:
		if len(t.Args) > 0 {
			return "swagger:" + t.Text + " " + strings.Join(t.Args, " ")
		}
		return "swagger:" + t.Text
	case TokenEOF, TokenYAMLFence, TokenText:
		return t.Text
	default:
		return t.Text
	}
}
