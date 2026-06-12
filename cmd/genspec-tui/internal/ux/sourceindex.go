// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"sort"
	"strings"

	"github.com/go-openapi/codescan/internal/scanner"
)

// SourceIndex is the caller-owned source-side half of the cross-ref linker
// (design §3): it maps the RFC 6901 JSON pointers codescan emits via
// OnProvenance to the Go source position that produced them, and back. codescan
// anchors only code-detail nodes, so this is NOT a bijection — a finer pointer
// resolves to its nearest anchored ancestor (PositionFor), and a source line
// resolves to its nearest enclosing anchor (PointerAt).
type SourceIndex struct {
	fwd    map[string]token.Position // pointer → source position
	byFile map[string][]lineAnchor   // absolute file → anchors sorted by line
}

// lineAnchor is one (1-based source line → pointer) entry within a file, for the
// reverse source→spec lookup.
type lineAnchor struct {
	line int
	ptr  string
}

// BuildSourceIndex builds the index from the provenance records collected during
// a scan (one OnProvenance call each). Later records win on a duplicate pointer
// (upsert / last-wins), matching codescan's build, where a node may be rewritten.
func BuildSourceIndex(provs []scanner.Provenance) *SourceIndex {
	x := &SourceIndex{
		fwd:    make(map[string]token.Position, len(provs)),
		byFile: make(map[string][]lineAnchor),
	}
	for _, p := range provs {
		x.fwd[p.Pointer] = p.Pos
		if p.Pos.Filename != "" {
			x.byFile[p.Pos.Filename] = append(x.byFile[p.Pos.Filename], lineAnchor{line: p.Pos.Line, ptr: p.Pointer})
		}
	}
	for f := range x.byFile {
		anchors := x.byFile[f]
		sort.Slice(anchors, func(i, j int) bool { return anchors[i].line < anchors[j].line })
	}
	return x
}

// Len reports how many anchored pointers the index holds (0 for a nil index).
func (x *SourceIndex) Len() int {
	if x == nil {
		return 0
	}
	return len(x.fwd)
}

// PositionFor returns the source position anchored to ptr, or — when ptr itself
// is a finer node with no anchor of its own — the position of its nearest
// anchored ancestor. The walk trims one pointer segment at a time (a zero-alloc
// suffix shrink), so /definitions/User/properties/x/items resolves to
// /definitions/User/properties/x, then /definitions/User, … until a hit.
func (x *SourceIndex) PositionFor(ptr string) (token.Position, bool) {
	if x == nil {
		return token.Position{}, false
	}
	for ptr != "" {
		if pos, ok := x.fwd[ptr]; ok {
			return pos, true
		}
		i := strings.LastIndexByte(ptr, '/')
		if i < 0 {
			break
		}
		ptr = ptr[:i]
	}
	return token.Position{}, false
}

// FirstAnchor returns the pointer of the earliest (lowest-line) anchor recorded
// in file, or ok=false when the file produced no spec node. Backs the tree's
// "locate this file in the spec" jump.
func (x *SourceIndex) FirstAnchor(file string) (string, bool) {
	if x == nil {
		return "", false
	}
	anchors := x.byFile[file]
	if len(anchors) == 0 {
		return "", false
	}
	return anchors[0].ptr, true // byFile is sorted by line
}

// PointerAt returns the pointer of the nearest anchor at or above (file, line) —
// the spec node enclosing that source line. line is 1-based (token.Position.Line).
// The bool is false when the file holds no anchors or line precedes the first.
func (x *SourceIndex) PointerAt(file string, line int) (string, bool) {
	if x == nil {
		return "", false
	}
	anchors := x.byFile[file]
	if len(anchors) == 0 {
		return "", false
	}
	// greatest anchor line <= line
	i := sort.Search(len(anchors), func(i int) bool { return anchors[i].line > line }) - 1
	if i < 0 {
		return "", false
	}
	return anchors[i].ptr, true
}
