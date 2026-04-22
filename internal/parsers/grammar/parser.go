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
	"slices"
	"strconv"
	"strings"
)

// Parser is the consumer contract the analyzer layer (bridge-taggers)
// depends on. The grammar package ships *DefaultParser as the single
// concrete implementation; the interface exists so tests can
// substitute a mock that distributes fabricated Blocks without
// running the grammar pipeline.
//
// This is the unlock for P5's property-based builder tests
// (architecture §5.3): tests construct Block values directly and
// feed them through a mock Parser — no string-formatting of comments
// and no re-parsing.
//
// Production code takes the concrete *DefaultParser returned by
// NewParser; only test code (and the rare case needing a mock)
// depends on the interface.
type Parser interface {
	// Parse runs the full preprocess → lex → parse pipeline on a
	// comment group and returns the typed Block that describes it.
	// Never panics; diagnostics accumulate on the returned Block.
	Parse(cg *ast.CommentGroup) Block

	// ParseText parses raw comment content (markers already stripped
	// by the caller). Used by LSP — the editor provides the raw text
	// at cursor position — and by tests synthesising input.
	ParseText(text string, pos token.Position) Block

	// ParseAs forces the annotation kind and parses the body under
	// it. Useful for LSP completion where the annotation line is
	// missing or being typed: given "the user is editing a model
	// block", parse the properties under the assumed kind. (See
	// architecture §4.6.)
	ParseAs(kind AnnotationKind, text string, pos token.Position) Block
}

// DefaultParser is the grammar package's concrete Parser
// implementation. Safe for concurrent use across goroutines.
// Constructed via NewParser.
type DefaultParser struct {
	fset           *token.FileSet
	diagnosticSink func(Diagnostic)
}

// NewParser constructs a DefaultParser bound to a FileSet (needed to
// map ast.CommentGroup positions to absolute source positions).
// Returns the concrete *DefaultParser so callers get full IDE
// discoverability; the Parser interface is the seam tests use to
// inject a mock.
//
// Variadic Options tune behavior — see WithDiagnosticSink. A
// zero-option call is the common case.
func NewParser(fset *token.FileSet, opts ...Option) *DefaultParser {
	p := &DefaultParser{fset: fset}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Option configures a DefaultParser built with NewParser.
type Option func(*DefaultParser)

// WithDiagnosticSink sets an optional callback invoked for every
// Diagnostic the parser emits, in addition to accumulating it on
// the returned Block. Useful for LSP streaming where diagnostics
// must be surfaced as they are produced, not batched until parse
// completes. The sink runs on the parser's goroutine; callers
// needing async delivery should push into a channel.
func WithDiagnosticSink(sink func(Diagnostic)) Option {
	return func(p *DefaultParser) { p.diagnosticSink = sink }
}

//nolint:ireturn // see Parse godoc
func (p *DefaultParser) Parse(cg *ast.CommentGroup) Block {
	lines := Preprocess(cg, p.fset)
	tokens := Lex(lines)
	return p.runParser(tokens)
}

//nolint:ireturn // see Parse godoc
func (p *DefaultParser) ParseText(text string, pos token.Position) Block {
	lines := preprocessText(text, pos)
	tokens := Lex(lines)
	return p.runParser(tokens)
}

//nolint:ireturn // see Parse godoc
func (p *DefaultParser) ParseAs(kind AnnotationKind, text string, pos token.Position) Block {
	// Prepend a synthetic annotation line so the parser dispatches to
	// the requested kind. If text already contains a swagger:<name>
	// annotation, the existing line wins (findAnnotation picks the
	// first) — the injected line is effectively decorative.
	injected := "swagger:" + kind.String() + "\n" + text
	return p.ParseText(injected, pos)
}

// runParser constructs a parseState wired with the DefaultParser's
// options (diagnostic sink, future additions) and delegates.
//
//nolint:ireturn // see Parse godoc
func (p *DefaultParser) runParser(tokens []Token) Block {
	ps := &parseState{tokens: tokens, sink: p.diagnosticSink}
	return ps.parse()
}

// Parse runs the full preprocess → lex → parse pipeline on a comment
// group and returns the typed Block that describes it. Never panics;
// diagnostics accumulate on the returned Block.
//
// A nil CommentGroup produces an empty UnboundBlock — useful for code
// paths that call Parse unconditionally.
//
// Convenience wrapper around NewParser(fset).Parse(cg) — preferred
// for one-off uses; store the Parser and reuse it for batch work.
//
//nolint:ireturn // see Parse godoc
func Parse(cg *ast.CommentGroup, fset *token.FileSet) Block {
	return NewParser(fset).Parse(cg)
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

// preprocessText converts raw text (already stripped of Go comment
// markers) into a []Line. Used by ParseText/ParseAs where no
// *ast.CommentGroup is available.
func preprocessText(text string, basePos token.Position) []Line {
	rawLines := strings.Split(text, "\n")
	out := make([]Line, 0, len(rawLines))
	for i, r := range rawLines {
		pos := basePos
		pos.Line += i
		if i > 0 {
			pos.Column = 1
		}
		out = append(out, Line{Text: r, Raw: r, Pos: pos})
	}
	return out
}

type parseState struct {
	tokens []Token
	diag   []Diagnostic
	sink   func(Diagnostic)
}

// emit records a diagnostic: appends to the block-local slice AND
// (if configured) pushes to the optional sink for streaming.
func (p *parseState) emit(d Diagnostic) {
	if p.sink != nil {
		p.sink(d)
	}
	p.diag = append(p.diag, d)
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
		// When the annotation trails the block (e.g., a `swagger:meta`
		// at the tail of a package doc), body tokens live in the
		// pre-annotation slice. Split pre at the first body-start so
		// Title/Description collection stops before structural
		// tokens, and those tokens reach parseBody alongside any
		// post-annotation ones.
		if splitIdx := findBodyStart(pre); splitIdx >= 0 {
			tail := pre[splitIdx:]
			pre = pre[:splitIdx]
			post = append(append([]Token{}, tail...), post...)
		}
	} else {
		base = newBaseBlock(AnnUnknown, firstMeaningfulPos(p.tokens))
		typed = &UnboundBlock{baseBlock: base}
		// For UnboundBlock (no annotation, e.g., a struct-field
		// docstring), split at the first body token so the prose
		// prelude still becomes Title/Description. Without this,
		// `// Name of the user.\n// required: true` loses the
		// docstring entirely.
		if splitIdx := findBodyStart(p.tokens); splitIdx >= 0 {
			pre = p.tokens[:splitIdx]
			post = p.tokens[splitIdx:]
		} else {
			pre = p.tokens
		}
	}

	p.parseTitleDesc(base, pre)
	p.parseBody(base, post)
	p.checkContextValidity(base)

	base.diagnostics = append(base.diagnostics, p.diag...)
	return typed
}

// checkContextValidity emits CodeContextInvalid warnings for every
// Property whose keyword is not legal under the block's
// AnnotationKind. Non-fatal (SeverityWarning); the analyzer decides
// policy. Skipped for UnboundBlock and non-dispatched annotations
// where context legality isn't meaningful at the parser layer.
func (p *parseState) checkContextValidity(base *baseBlock) {
	allowed := allowedContexts(base.kind)
	if allowed == nil {
		return
	}
	for _, prop := range base.properties {
		if contextsOverlap(prop.Keyword.Contexts, allowed) {
			continue
		}
		p.emit(Warnf(prop.Pos, CodeContextInvalid,
			"keyword %q not valid under swagger:%s (legal in: %s)",
			prop.Keyword.Name, base.kind,
			formatKeywordContexts(prop.Keyword.Contexts)))
	}
}

// allowedContexts returns the set of Kind sub-contexts that may host
// keywords under the given AnnotationKind. Returns nil to mean "no
// parser-layer check" (UnboundBlock, strfmt, alias, etc., where the
// legality depends on external context the parser doesn't have).
//
// The sets are deliberately broad: an operation body can contain
// schema properties, response headers, parameters, and more, so
// allowedContexts(AnnOperation) lists all plausible sub-contexts.
// Analyzers may enforce tighter rules with more context (Go type,
// enclosing struct) but the parser uses the permissive union.
func allowedContexts(a AnnotationKind) []Kind {
	switch a {
	case AnnModel:
		return []Kind{KindSchema, KindItems}
	case AnnParameters:
		return []Kind{KindParam, KindSchema, KindItems}
	case AnnResponse:
		return []Kind{KindResponse, KindSchema, KindHeader, KindItems}
	case AnnOperation:
		return []Kind{KindOperation, KindParam, KindSchema, KindHeader, KindItems, KindResponse}
	case AnnRoute:
		return []Kind{KindRoute, KindParam, KindSchema, KindHeader, KindItems, KindResponse}
	case AnnMeta:
		return []Kind{KindMeta, KindSchema}
	case AnnUnknown,
		AnnStrfmt, AnnAlias, AnnName, AnnAllOf, AnnEnumDecl,
		AnnIgnore, AnnDefaultName, AnnType, AnnFile:
		return nil
	default:
		return nil
	}
}

// contextsOverlap reports whether any Kind in the keyword's contexts
// list is in the allowed set.
func contextsOverlap(kwContexts []ContextDoc, allowed []Kind) bool {
	for _, cd := range kwContexts {
		if slices.Contains(allowed, cd.Kind) {
			return true
		}
	}
	return false
}

// formatKeywordContexts renders a keyword's legal Kind list for
// diagnostics — "schema, param, items".
func formatKeywordContexts(ctxs []ContextDoc) string {
	if len(ctxs) == 0 {
		return "(none)"
	}
	out := make([]string, len(ctxs))
	for i, c := range ctxs {
		out[i] = c.Kind.String()
	}
	return strings.Join(out, ", ")
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

// findBodyStart returns the index of the first "body" token — a
// keyword, YAML fence, or raw YAML line — or -1 if the stream is
// entirely prose (TEXT + BLANK + EOF). Used to split UnboundBlock
// tokens into a Title/Description prelude and a property body.
func findBodyStart(tokens []Token) int {
	for i, t := range tokens {
		switch t.Kind {
		case TokenKeywordValue, TokenKeywordBlockHead,
			TokenYAMLFence, TokenRawLine:
			return i
		case TokenEOF, TokenBlank, TokenText, TokenAnnotation:
			// Prose / control tokens — keep scanning.
		default:
			// Future kinds: err on the side of treating them as
			// body so analyzers notice.
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
		p.emit(Errorf(tok.Pos, CodeInvalidAnnotation,
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
// appear before the annotation. It also accumulates the raw prose
// lines (source-order, with blank separators preserved) so consumers
// can reproduce v1's SectionedParser.header — see baseBlock.ProseLines.
//
// Keyword/YAML/block-head tokens appearing pre-annotation are unusual
// but not fatal — they are ignored with no diagnostic for v1 parity.
func (p *parseState) parseTitleDesc(base *baseBlock, pre []Token) {
	var paragraphs []string
	var current []string
	var proseLines []string

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
			proseLines = append(proseLines, "")
		case TokenText:
			current = append(current, t.Text)
			proseLines = append(proseLines, t.Text)
		case TokenEOF,
			TokenAnnotation,
			TokenKeywordValue, TokenKeywordBlockHead,
			TokenYAMLFence, TokenRawLine:
			// Ignored in the title/description slice. Structural
			// tokens are routed through parseBody via parse()'s
			// body-split even when the annotation is trailing.
		default:
			// Unreachable at v1; future kinds ignored defensively.
		}
	}
	flush()

	// Trim trailing blank lines from proseLines so JoinDropLast
	// produces parity with v1's SectionedParser, which stopped
	// collecting header lines at the first tagger match and never
	// accumulated more than one trailing blank.
	for len(proseLines) > 0 && proseLines[len(proseLines)-1] == "" {
		proseLines = proseLines[:len(proseLines)-1]
	}
	base.proseLines = proseLines

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
		case TokenEOF, TokenBlank, TokenText, TokenRawLine:
			// TokenRawLine outside a fence shouldn't happen (lexer
			// tracks fence state) — ignore defensively.
			i++

		case TokenKeywordValue:
			base.properties = append(base.properties, Property{
				Keyword:    *t.Keyword,
				Pos:        t.Pos,
				Value:      t.Value,
				Typed:      p.typeConvert(*t.Keyword, t.Value, t.Pos),
				ItemsDepth: t.ItemsDepth,
			})
			i++

		case TokenKeywordBlockHead:
			i = p.collectBlockBody(base, post, i)

		case TokenYAMLFence:
			i = p.collectYAMLBody(base, post, i)

		case TokenAnnotation:
			p.emit(Warnf(t.Pos, CodeInvalidAnnotation,
				"additional swagger:%s annotation ignored (one per comment block)",
				t.Text))
			i++

		default:
			i++
		}
	}
}

// collectBlockBody emits a Property for the KEYWORD_BLOCK_HEAD token
// at index i and consumes any subsequent TEXT tokens as the block's
// Body. Collection stops (per legacy S6 "multi-line tagger switch")
// at the next structured token — another keyword, annotation, YAML
// fence, or EOF. Blank tokens are treated as body-internal separators
// if followed by more text; a trailing run of blanks is trimmed.
//
// When the block-head keyword is "extensions" or "infoExtensions",
// each body line of the form `name: value` is *also* emitted as a
// top-level Extension on the Block so `block.Extensions()` exposes
// them uniformly. The original Body is still populated.
//
// Raw-block absorption: when the head's ValueType is RawBlock
// (consumes/produces/security/parameters/responses/extensions at a
// route or operation level), scalar keyword-shaped body lines
// (RawValue-typed keywords like `default:` or `example:`) are
// absorbed as body text rather than treated as terminators. Mirrors
// v1's SectionedParser behavior: without a registered top-level
// tagger for `default`, `example`, etc., those lines fall through
// into the currently-active multi-line tagger's body. Terminators
// (other block heads, single-line route-structural keywords like
// `schemes:` with ValueType = CommaList, and `deprecated:` Boolean)
// still stop collection.
//
// Returns the index past the last body token consumed.
func (p *parseState) collectBlockBody(base *baseBlock, post []Token, i int) int {
	head := post[i]
	prop := Property{
		Keyword:    *head.Keyword,
		Pos:        head.Pos,
		ItemsDepth: head.ItemsDepth,
	}
	i++

	isExtensions := isExtensionBlock(head.Keyword.Name)
	isRawBlock := head.Keyword.Value.Type == ValueRawBlock

	var pendingBlanks int
	for i < len(post) {
		next := post[i]
		switch next.Kind {
		case TokenEOF,
			TokenAnnotation,
			TokenKeywordBlockHead,
			TokenRawLine:
			base.properties = append(base.properties, prop)
			return i
		case TokenYAMLFence:
			if isExtensions {
				// Extension blocks may wrap their body in `---`
				// fences (e.g., `Extensions:\n---\nx-foo: false\n---`).
				// Absorb the fence-internal lines into prop.Body so
				// the downstream v1-style extension body parser sees
				// them. The closing fence is skipped.
				i = p.absorbFencedExtensionBody(base, &prop, post, i)
				continue
			}
			base.properties = append(base.properties, prop)
			return i
		case TokenKeywordValue:
			if absorbed := p.absorbRawBlockKeyword(&prop, next, isRawBlock, &pendingBlanks); !absorbed {
				base.properties = append(base.properties, prop)
				return i
			}
		case TokenText:
			p.appendRawBlockText(base, &prop, next, isExtensions, &pendingBlanks)
		case TokenBlank:
			// Defer — include only if more text follows within the
			// block. Trailing blanks are dropped.
			pendingBlanks++
		default:
			// Defensive: unknown future token kinds end the body.
			base.properties = append(base.properties, prop)
			return i
		}
		i++
	}

	base.properties = append(base.properties, prop)
	return i
}

// absorbFencedExtensionBody consumes the `---`-fenced body of an
// extensions block, appending each interior line (as Raw) to
// prop.Body and emitting each as an Extension entry via
// parseExtensionLine. Returns the index past the closing fence (or
// past EOF if the block is unterminated). Mirrors how v1's
// SetOpExtensions.Parse treats fence-wrapped extension bodies — the
// fences are consumed, the interior is parsed with indentation
// intact.
func (p *parseState) absorbFencedExtensionBody(base *baseBlock, prop *Property, post []Token, i int) int {
	openerPos := post[i].Pos
	i++
	for i < len(post) {
		next := post[i]
		if next.Kind == TokenYAMLFence {
			return i + 1 // skip closing fence
		}
		if next.Kind == TokenEOF {
			p.emit(Errorf(openerPos, CodeUnterminatedYAML,
				"YAML body opened with --- but never closed"))
			return i
		}
		if next.Kind == TokenRawLine {
			prop.Body = append(prop.Body, next.Text)
			// Also extract Extension entries. parseExtensionLine
			// expects a TEXT-shaped Token — synthesise one with the
			// raw line.
			surrogate := Token{Kind: TokenText, Text: strings.TrimSpace(next.Text), Pos: next.Pos}
			if ext, ok := parseExtensionLine(surrogate); ok {
				if isExtensionName(ext.Name) {
					base.extensions = append(base.extensions, ext)
				}
			}
		}
		i++
	}
	return i
}

// absorbRawBlockKeyword handles a TokenKeywordValue encountered
// inside a RawBlock body. Returns true iff the token was absorbed as
// body text (mirroring v1's tagger-based collection where sub-
// context keywords like `default:` / `in:` / `max:` fall through
// into the active multi-line tagger's body). Returns false when the
// token is a legitimate sibling terminator (route/operation/meta
// structural keyword) that should end the block.
func (p *parseState) absorbRawBlockKeyword(prop *Property, next Token, isRawBlock bool, pendingBlanks *int) bool {
	if !isRawBlock || next.Keyword == nil || isRouteStructuralKeyword(next.Keyword) {
		return false
	}
	for range *pendingBlanks {
		prop.Body = append(prop.Body, "")
	}
	*pendingBlanks = 0
	name := next.SourceName
	if name == "" {
		name = next.Keyword.Name
	}
	prop.Body = append(prop.Body, name+": "+next.Value)
	return true
}

// appendRawBlockText accumulates a TokenText line into prop.Body,
// using the indentation-preserving Raw form for extensions bodies
// (where nested YAML-like maps rely on source indentation) and the
// cleaned Text form for other raw-block bodies. When the head is an
// extensions block, the line is also parsed into an Extension entry
// for the block-level accessor.
func (p *parseState) appendRawBlockText(base *baseBlock, prop *Property, next Token, isExtensions bool, pendingBlanks *int) {
	for range *pendingBlanks {
		prop.Body = append(prop.Body, "")
	}
	*pendingBlanks = 0
	body := next.Text
	if isExtensions && next.Raw != "" {
		body = next.Raw
	}
	prop.Body = append(prop.Body, body)
	if !isExtensions {
		return
	}
	ext, ok := parseExtensionLine(next)
	if !ok {
		return
	}
	if !isExtensionName(ext.Name) {
		p.emit(Warnf(ext.Pos, CodeInvalidExtension,
			"extension name %q must begin with 'x-' or 'X-'", ext.Name))
	}
	base.extensions = append(base.extensions, ext)
}

// isExtensionBlock reports whether the given keyword name declares an
// extensions block (i.e., `extensions:` or `infoExtensions:`).
func isExtensionBlock(name string) bool {
	return name == "extensions" || name == "infoExtensions"
}

// isRouteStructuralKeyword reports whether kw is a top-level
// keyword at the route/operation/meta annotation level — the set
// that v1's SectionedParser registers as taggers at those
// annotations and that terminates the currently-active multi-line
// tagger's body. A keyword qualifies if any of its declared contexts
// is KindRoute, KindOperation, or KindMeta.
//
// Used by collectBlockBody to decide, in a RawBlock body, whether an
// incoming TokenKeywordValue is a sibling top-level tag (terminate)
// or a sub-context keyword (`in:`, `required:`, `default:`) that
// legitimately appears as body text inside `Parameters:` /
// `Responses:` block entries.
func isRouteStructuralKeyword(kw *Keyword) bool {
	for _, ctx := range kw.Contexts {
		switch ctx.Kind {
		case KindRoute, KindOperation, KindMeta:
			return true
		case KindParam, KindSchema, KindHeader, KindItems, KindResponse:
			// Sub-contexts — keep scanning; a keyword can have
			// multiple contexts and any route/operation/meta match
			// wins.
		default:
			// Unknown/future kinds: conservative — don't terminate.
		}
	}
	return false
}

// isExtensionName reports whether s is a well-formed OpenAPI vendor
// extension name: it must begin with "x-" or "X-" and have at least
// one character after the hyphen. Mirrors the v1 rxAllowedExtensions
// check (`^[Xx]-`).
func isExtensionName(s string) bool {
	const minExtNameLen = 3 // "x-" + at least one suffix character
	if len(s) < minExtNameLen {
		return false
	}
	if (s[0] != 'x' && s[0] != 'X') || s[1] != '-' {
		return false
	}
	return true
}

// parseExtensionLine extracts `name: value` from a body TEXT token,
// returning an Extension with the token's Pos. Returns (zero, false)
// for lines that don't match the form. Name and Value are
// whitespace-trimmed; name-well-formedness (the `x-*` requirement)
// is a separate P2.4 check downstream.
func parseExtensionLine(t Token) (Extension, bool) {
	before, after, found := strings.Cut(t.Text, ":")
	if !found {
		return Extension{}, false
	}
	name := strings.TrimSpace(before)
	if name == "" {
		return Extension{}, false
	}
	return Extension{
		Name:  name,
		Value: strings.TrimSpace(after),
		Pos:   t.Pos,
	}, true
}

// collectYAMLBody captures everything between a YAML_FENCE opener at
// index i and its matching closer (or EOF). Emits an UnterminatedYAML
// diagnostic if no closer is found. Returns the index past the closer.
//
// Inside a fence the lexer emits TokenRawLine tokens carrying Line.Raw,
// so the body survives verbatim (indentation preserved) and can be
// handed directly to internal/parsers/yaml/ for further parsing.
func (p *parseState) collectYAMLBody(base *baseBlock, post []Token, i int) int {
	openerPos := post[i].Pos
	i++

	var body []string
	for i < len(post) && post[i].Kind != TokenYAMLFence && post[i].Kind != TokenEOF {
		body = append(body, post[i].Text)
		i++
	}

	if i < len(post) && post[i].Kind == TokenYAMLFence {
		i++ // consume closer
	} else {
		p.emit(Errorf(openerPos, CodeUnterminatedYAML,
			"YAML body opened with --- but never closed"))
	}

	base.yamlBlocks = append(base.yamlBlocks, RawYAML{
		Pos:  openerPos,
		Text: strings.Join(body, "\n"),
	})
	return i
}

// typeConvert performs primitive value-typing per the keyword's
// declared ValueType (architecture §3.4). Primitives (Number, Integer,
// Boolean, StringEnum) are converted at parse time and populate the
// corresponding TypedValue field. Non-primitive ValueTypes (String,
// CommaList, RawValue, RawBlock, None) return a zero TypedValue — the
// analyzer consumes the raw Property.Value with knowledge of the
// target Go type.
//
// Conversion failures emit non-fatal diagnostics on p.diag; the
// returned TypedValue stays zero so downstream consumers can tell
// "no conversion performed" from "conversion succeeded with zero
// value" via Typed.Type.
func (p *parseState) typeConvert(kw Keyword, raw string, pos token.Position) TypedValue {
	switch kw.Value.Type {
	case ValueNumber:
		op, rest := splitCmpOperator(raw)
		n, err := strconv.ParseFloat(strings.TrimSpace(rest), 64)
		if err != nil {
			p.emit(Errorf(pos, CodeInvalidNumber,
				"%s: %q is not a valid number", kw.Name, raw))
			return TypedValue{}
		}
		return TypedValue{Type: ValueNumber, Op: op, Number: n}

	case ValueInteger:
		i, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			p.emit(Errorf(pos, CodeInvalidInteger,
				"%s: %q is not a valid integer", kw.Name, raw))
			return TypedValue{}
		}
		return TypedValue{Type: ValueInteger, Integer: i}

	case ValueBoolean:
		b, ok := parseBool(raw)
		if !ok {
			p.emit(Errorf(pos, CodeInvalidBoolean,
				"%s: %q is not a valid boolean (expected true or false)", kw.Name, raw))
			return TypedValue{}
		}
		return TypedValue{Type: ValueBoolean, Boolean: b}

	case ValueStringEnum:
		for _, allowed := range kw.Value.Values {
			if strings.EqualFold(raw, allowed) {
				return TypedValue{Type: ValueStringEnum, String: allowed}
			}
		}
		p.emit(Errorf(pos, CodeInvalidStringEnum,
			"%s: %q is not one of {%s}",
			kw.Name, raw, strings.Join(kw.Value.Values, ", ")))
		return TypedValue{}

	case ValueNone, ValueString, ValueCommaList, ValueRawValue, ValueRawBlock:
		return TypedValue{}

	default:
		return TypedValue{}
	}
}

// splitCmpOperator strips a leading comparison operator ("<=", ">=",
// "<", ">", "=") from s, returning the operator (or "") and the rest.
// Supports the v1 `maximum: <5` / `minimum: >=0` forms.
func splitCmpOperator(s string) (op, rest string) {
	s = strings.TrimLeft(s, " \t")
	for _, candidate := range []string{"<=", ">=", "<", ">", "="} {
		if strings.HasPrefix(s, candidate) {
			return candidate, s[len(candidate):]
		}
	}
	return "", s
}

// parseBool accepts only "true" or "false" (case-insensitive). stdlib
// strconv.ParseBool is too lenient for the swagger grammar, accepting
// "1", "t", "T", etc.
func parseBool(s string) (bool, bool) {
	s = strings.TrimSpace(s)
	switch {
	case strings.EqualFold(s, "true"):
		return true, true
	case strings.EqualFold(s, "false"):
		return false, true
	default:
		return false, false
	}
}
