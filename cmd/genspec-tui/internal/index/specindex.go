// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"sort"

	lexer "github.com/go-openapi/core/json/lexers/default-lexer"
	yamllexer "github.com/go-openapi/core/json/lexers/yaml-lexer"
)

// SpecIndex maps between rendered-spec lines and the RFC 6901 JSON pointer of the spec node shown on each line.
//
// It is the spec-side half of the cross-ref linker: line ↔ pointer, built fresh from the exact bytes the spec pane displays.
// Lines are 0-based (matching the viewport's strings.Split addressing); the same structure also anchors spec remarks.
type SpecIndex struct {
	line2ptr map[int]string
	ptr2line map[string]int
	lines    []int // sorted keys of line2ptr, for nearest-preceding lookup
}

// NewSpecIndex finalizes the maps into a SpecIndex with sorted line keys.
func NewSpecIndex(line2ptr map[int]string, ptr2line map[string]int) *SpecIndex {
	lines := make([]int, 0, len(line2ptr))
	for l := range line2ptr {
		lines = append(lines, l)
	}
	sort.Ints(lines)
	return &SpecIndex{line2ptr: line2ptr, ptr2line: ptr2line, lines: lines}
}

// Indexes are the per-render products of a single lexer walk over the rendered spec.
//
// It tells where each node is, where each $ref points, and what every token is.
//
// They are grouped because they share the walk — adding a fourth should not mean a fourth traversal of the same bytes.
type Indexes struct {
	Spec      *SpecIndex
	Refs      *RefIndex
	Highlight *HighlightIndex
}

// BuildJSONIndex builds the per-render indexes from indented JSON bytes in ONE lexer pass.
//
// The lexer reports, per token, its JSON pointer (RFC 6901 escaping handled for us) and its 1-based source line.
//
// The first token to report a pointer is the line that member is declared on; later repeats (its value, its closing
// delimiter) are the same node seen again.
// Reads the bytes the pane renders, so the ordered keys of spec.Swagger's MarshalJSON are preserved.
func BuildJSONIndex(b []byte) Indexes {
	acc := newIndexAccum()

	lex := lexer.NewVerbatimWithBytes(b, lexer.WithJSONPointer(true))
	for tok := range lex.Tokens() {
		line := lex.Line() - 1
		acc.add(lex.JSONPointer().String(), line, tok.IsKey(), tok.Value())
		acc.addSpan(line, lex.Column(), tok.Kind())
	}

	return acc.finish()
}

// BuildYAMLIndex is BuildJSONIndex over the YAML render.
//
// The YAML lexer emits the same logical token stream with the same RFC 6901 pointer escaping, so the indexes built from
// either render are interchangeable — only the lines differ.
func BuildYAMLIndex(b []byte) Indexes {
	acc := newIndexAccum()

	lex := yamllexer.NewWithBytes(b, yamllexer.WithJSONPointer(true))
	for tok := range lex.Tokens() {
		line := lex.Line() - 1
		acc.add(lex.JSONPointer().String(), line, tok.IsKey(), tok.Value())
		acc.addSpan(line, lex.Column(), tok.Kind())
	}

	return acc.finish()
}

// PointerAt returns the JSON pointer of the node at line, or the nearest member line above it when line itself carries
// no pointer (e.g. a closing brace).
//
// The bool is false only for an empty index or a line above the first member.
func (x *SpecIndex) PointerAt(line int) (string, bool) {
	if x == nil || len(x.lines) == 0 {
		return "", false
	}
	if p, ok := x.line2ptr[line]; ok {
		return p, true
	}
	// greatest indexed line <= line
	i := sort.SearchInts(x.lines, line+1) - 1
	if i < 0 {
		return "", false
	}
	return x.line2ptr[x.lines[i]], true
}

// LineForPointer returns the 0-based line where pointer is rendered.
func (x *SpecIndex) LineForPointer(ptr string) (int, bool) {
	if x == nil {
		return 0, false
	}
	l, ok := x.ptr2line[ptr]
	return l, ok
}

// Len reports how many pointers the index holds (0 for a nil index).
func (x *SpecIndex) Len() int {
	if x == nil {
		return 0
	}
	return len(x.ptr2line)
}
