// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package godoclink rewrites godoc-specific syntax that reads as noise when a
// Go doc comment is carried into a Swagger title / description (the
// Options.CleanGoDoc feature).
//
// Phase 1 (this package as it stands) covers the resolution-free transforms:
//
//   - reference-style link definition lines (`[text]: url`) are dropped;
//   - godoc doc-link spans (`[Widget]`, `[pkg.Type]`, `[Order.Field]`) have
//     their brackets removed and the (leaf) identifier humanized via the swag
//     name mangler — e.g. `[CustName]` → "cust name";
//   - the first identifier of the prose is restored to sentence case.
//
// A later phase adds identifier→schema resolution so a doc-link that points at
// an emitted definition is recomposed to that definition's exposed name rather
// than merely humanized. The recognizer regexes are adapted from the
// battle-tested github.com/fredbi/go-fred-mcp/pkg/doc-filters/godoc-filter; the
// key difference is that this package *rewrites* the prose whereas that tool
// *redacts* (length-preserving blanking) for masking.
//
// See .claude/plans/features/godoc-filter-design.md.
package godoclink

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-openapi/swag/mangling"
)

// docLinkRE matches a godoc doc-link span: a dotted identifier chain
// (`[pkg.Type]`, `[Order.Field]`) OR an uppercase-led single (`[Widget]`),
// tolerating a leading `*` (`[*time.Time]`). Bare-lowercase singles (`[id]`),
// empty (`[]byte`), digit-led (`[0]`) and multi-word (`[see notes]`) spans are
// deliberately not matched, so ordinary prose brackets are left intact.
var docLinkRE = regexp.MustCompile(`\[\*?\w+(?:\.\w+)+\]|\[\*?[A-Z]\w*\]`)

// refDefLineRE matches a reference-style link definition line — `[text]: url`
// on its own line (optionally indented). Such a line is godoc/markdown link
// plumbing and carries no prose, so the whole line is dropped.
var refDefLineRE = regexp.MustCompile(`^[ \t]*\[[^\]]+\]:[ \t]+\S.*$`)

// multiNewlineRE collapses a 3+ newline run (left behind by dropping an
// interior reference-definition line) back to a single paragraph break.
var multiNewlineRE = regexp.MustCompile(`\n{3,}`)

// Clean applies the resolution-free godoc filtering to text using m for
// identifier humanization. It is the caller's responsibility to apply Clean
// ONLY to godoc-derived prose — never to author-written swagger:title /
// swagger:description override text. Clean never mutates m.
func Clean(text string, m *mangling.NameMangler) string {
	if text == "" {
		return text
	}

	text = dropRefDefLines(text)
	text = rewriteDocLinks(text, m)

	return text
}

// dropRefDefLines removes reference-style link definition lines, collapsing the
// blank runs a mid-text drop would otherwise leave and trimming trailing space.
func dropRefDefLines(text string) string {
	if !strings.Contains(text, "]:") {
		return text
	}

	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	dropped := false
	for _, ln := range lines {
		if refDefLineRE.MatchString(ln) {
			dropped = true
			continue
		}
		kept = append(kept, ln)
	}
	if !dropped {
		return text
	}

	out := strings.Join(kept, "\n")
	// A dropped interior line can leave a 3+ newline run; fold to a paragraph
	// break. Trailing whitespace/newlines from a dropped final line are trimmed.
	out = multiNewlineRE.ReplaceAllString(out, "\n\n")

	return strings.TrimRight(out, " \t\n")
}

// rewriteDocLinks strips the brackets off each doc-link span and humanizes the
// leaf identifier; a span at the start of the prose is sentence-cased.
func rewriteDocLinks(text string, m *mangling.NameMangler) string {
	locs := docLinkRE.FindAllStringIndex(text, -1)
	if locs == nil {
		return text
	}

	var b strings.Builder
	b.Grow(len(text))
	last := 0
	for _, loc := range locs {
		b.WriteString(text[last:loc[0]])

		span := text[loc[0]:loc[1]]
		human := m.ToHumanNameLower(leafIdent(span))
		if isSentenceStart(text, loc[0]) {
			human = capitalizeFirst(human)
		}
		b.WriteString(human)

		last = loc[1]
	}
	b.WriteString(text[last:])

	return b.String()
}

// leafIdent extracts the trailing identifier of a doc-link span: it strips the
// surrounding brackets and an optional leading `*`, then keeps the last
// dot-separated segment (`[*inventory.Ledger]` → "Ledger").
func leafIdent(span string) string {
	inner := strings.TrimPrefix(span[1:len(span)-1], "*")
	if i := strings.LastIndex(inner, "."); i >= 0 {
		return inner[i+1:]
	}

	return inner
}

// isSentenceStart reports whether pos is the first non-space byte of text — the
// sentence-initial position where a substituted name is restored to title case.
func isSentenceStart(text string, pos int) bool {
	for i := range pos {
		if !unicode.IsSpace(rune(text[i])) {
			return false
		}
	}

	return true
}

// capitalizeFirst upper-cases the first rune of s, leaving the rest untouched.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}

	return string(unicode.ToUpper(r)) + s[size:]
}
