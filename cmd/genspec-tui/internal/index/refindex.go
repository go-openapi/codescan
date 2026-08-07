// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"net/url"
	"sort"
	"strings"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// refSuffix is the pointer tail a $ref member carries.
//
// Both lexers report the member's own pointer (.../boss/$ref) for the key AND for its value token.
const refSuffix = "/$ref"

// RefTarget is a parsed $ref value.
//
// Local refs point inside the document being rendered and can therefore be followed with the SpecIndex.
// Anything else (a sibling file, a URL, a bare filename) is recorded honestly but is not resolvable here.
// The TUI renders one spec, it is not a $ref resolver.
type RefTarget struct {
	Raw     string // exactly as written in the document
	Pointer string // the RFC 6901 pointer for a local ref; "" otherwise
	Local   bool   // whether the ref points inside this document
}

// RefSite is one place in the rendered spec where a $ref appears.
type RefSite struct {
	Pointer string // the node HOLDING the $ref (the /$ref segment trimmed)
	Line    int    // 0-based rendered line of the $ref
	Target  RefTarget
}

// RefIndex records every $ref in a rendered spec, keyed by what it points at.
//
// This is the "find references" half: a decl is anchored to its own field, never to the type it references,
// so answering "where is this definition used?" means resolving $refs at RENDER time.
//
// That makes the index per-render, exactly like SpecIndex - and it is built in the same lexer pass, so it costs no
// extra walk.
//
// Scope, deliberately: this is a SITE index, not a JSON-Schema resolver.
// It records where each $ref token sits and what string it holds.
// It does not follow ref-to-ref chains, does not reason about $refs nested in allOf, and does not apply the "sibling
// keywords are ignored" rule - $ref quirks stay documented but are not chased here.
type RefIndex struct {
	byTarget map[string][]RefSite // local target pointer → sites, ordered by line
	byLine   map[int]RefSite      // rendered line → the $ref on it
	total    int
}

// ParseRefTarget parses a raw $ref value.
//
// A local ref is a bare fragment ("#/definitions/User").
// The fragment is percent-DECODED, because go-openapi/spec marshals refs through net/url and will escape a definition
// name containing e.g. a space - while the SpecIndex keys on the document's own key text, which is not escaped.
// Decoding here is what makes the two sides comparable.
//
// A malformed escape falls back to the verbatim fragment rather than dropping the ref.
func ParseRefTarget(raw string) RefTarget {
	t := RefTarget{Raw: raw}
	if !strings.HasPrefix(raw, "#") {
		return t // another document, a URL, or a bare filename
	}
	frag := strings.TrimPrefix(raw, "#")
	if frag != "" && !strings.HasPrefix(frag, "/") {
		return t // "#Foo" - a plain-name fragment, not a JSON pointer
	}
	decoded, err := url.PathUnescape(frag)
	if err != nil {
		decoded = frag
	}
	t.Pointer, t.Local = decoded, true

	return t
}

// RefsToPointer returns every site referencing the node at ptr, ordered by rendered line.
//
// ptr is a plain JSON pointer as the SpecIndex reports it (e.g. "/definitions/User").
// The leading "#" of the $ref is not included:
// generated specs only produce $ref's as fragments rooted in the same document.
func (x *RefIndex) RefsToPointer(ptr string) []RefSite {
	if x == nil {
		return nil
	}

	return x.byTarget[ptr]
}

// RefsNear returns the sites referencing ptr.
//
// When nothing references ptr itself, it returns the sites referencing its nearest referenced ANCESTOR,
// along with the pointer that actually matched.
//
// The segment-trim walk mirrors SourceIndex.PositionFor, and for the same reason: the user's cursor is rarely on the
// definition line itself.
//
// Asking for the references of /definitions/User/properties/name should find the uses of User, not report nothing.
func (x *RefIndex) RefsNear(ptr string) (string, []RefSite) {
	if x == nil {
		return "", nil
	}
	for ptr != "" {
		if sites := x.byTarget[ptr]; len(sites) > 0 {
			return ptr, sites
		}
		i := strings.LastIndexByte(ptr, '/')
		if i < 0 {
			break
		}
		ptr = ptr[:i]
	}

	return "", nil
}

// RefAt returns the $ref rendered on the given 0-based line, if any.
//
// Backs go-to-definition: the user puts the cursor on a $ref and follows it.
func (x *RefIndex) RefAt(line int) (RefSite, bool) {
	if x == nil {
		return RefSite{}, false
	}
	site, ok := x.byLine[line]

	return site, ok
}

// LocalRefLines returns the 0-based rendered lines holding a FOLLOWABLE $ref, i.e. one pointing inside this document.
//
// Backs the spec pane's gutter: an external ref is not marked, because Enter cannot take you there.
func (x *RefIndex) LocalRefLines() []int {
	if x == nil {
		return nil
	}
	out := make([]int, 0, len(x.byLine))
	for line, site := range x.byLine {
		if site.Target.Local {
			out = append(out, line)
		}
	}

	return out
}

// Len reports how many $ref sites the index holds (0 for a nil index), including non-local ones.
func (x *RefIndex) Len() int {
	if x == nil {
		return 0
	}

	return x.total
}

// indexAccum accumulates both per-render indexes during one lexer walk.
//
// The two lexers (JSON and YAML) emit different concrete types but the same logical stream, so each drives this from
// its own loop.
type indexAccum struct {
	line2ptr map[int]string
	ptr2line map[string]int
	byTarget map[string][]RefSite
	byLine   map[int]RefSite
	total    int
	spans    map[int][]theme.Span
}

func newIndexAccum() *indexAccum {
	return &indexAccum{
		line2ptr: make(map[int]string),
		ptr2line: make(map[string]int),
		byTarget: make(map[string][]RefSite),
		byLine:   make(map[int]RefSite),
		spans:    make(map[int][]theme.Span),
	}
}

// add folds one token into both indexes.
//
// ptr is the token's JSON pointer and line its 0-based rendered line. isKey says whether it is an object key,
// and value carries its raw scalar text, empty for a delimiter.
func (a *indexAccum) add(ptr string, line int, isKey bool, value []byte) {
	if ptr == "" {
		return
	}

	// Spec index: the FIRST token to report a pointer is the line that pointer is declared on; later repeats (its value,
	// its closing delimiter) are the same node seen again.
	if _, seen := a.ptr2line[ptr]; !seen {
		a.ptr2line[ptr] = line
		a.line2ptr[line] = ptr
	}

	// Ref index: the VALUE token under a .../$ref pointer carries the target.
	// The key token shares that pointer, hence the isKey guard.
	if isKey || len(value) == 0 || !strings.HasSuffix(ptr, refSuffix) {
		return
	}
	site := RefSite{
		Pointer: strings.TrimSuffix(ptr, refSuffix),
		Line:    line,
		Target:  ParseRefTarget(string(value)),
	}
	a.total++
	a.byLine[line] = site
	if site.Target.Local {
		a.byTarget[site.Target.Pointer] = append(a.byTarget[site.Target.Pointer], site)
	}
}

func (a *indexAccum) finish() Indexes {
	for target := range a.byTarget {
		sites := a.byTarget[target]
		sort.Slice(sites, func(i, j int) bool { return sites[i].Line < sites[j].Line })
	}

	return Indexes{
		Spec:      NewSpecIndex(a.line2ptr, a.ptr2line),
		Refs:      &RefIndex{byTarget: a.byTarget, byLine: a.byLine, total: a.total},
		Highlight: &HighlightIndex{byLine: a.spans},
	}
}
