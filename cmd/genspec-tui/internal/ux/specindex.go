// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"bytes"
	"encoding/json/jsontext"
	"sort"
	"strconv"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// SpecIndex maps between rendered-spec lines and the RFC 6901 JSON pointer of
// the spec node shown on each line. It is the spec-side half of the cross-ref
// linker (design §4): line ↔ pointer, built fresh from the exact bytes the spec
// pane displays. Lines are 0-based (matching the viewport's strings.Split
// addressing); the same structure also anchors spec remarks.
type SpecIndex struct {
	line2ptr map[int]string
	ptr2line map[string]int
	lines    []int // sorted keys of line2ptr, for nearest-preceding lookup
}

// PointerAt returns the JSON pointer of the node at line, or the nearest member
// line above it when line itself carries no pointer (e.g. a closing brace). The
// bool is false only for an empty index or a line above the first member.
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

// newSpecIndex finalizes the maps into a SpecIndex with sorted line keys.
func newSpecIndex(line2ptr map[int]string, ptr2line map[string]int) *SpecIndex {
	lines := make([]int, 0, len(line2ptr))
	for l := range line2ptr {
		lines = append(lines, l)
	}
	sort.Ints(lines)
	return &SpecIndex{line2ptr: line2ptr, ptr2line: ptr2line, lines: lines}
}

// BuildJSONIndex builds a SpecIndex from indented JSON bytes using the
// jsontext.Decoder token stream: after each token, StackPointer() names the
// current node, and InputOffset() locates it. The first occurrence of a pointer
// is the member's declaration line; later repeats (its value, its close) are
// ignored. Best-effort — on a decode error it returns what it has so far (the
// rendered bytes are always valid JSON, so this is just defensive).
//
// Requires GOEXPERIMENT=jsonv2 (see plan §4 / build §A0).
func BuildJSONIndex(b []byte) *SpecIndex {
	lt := newLineTable(b)
	line2ptr := make(map[int]string)
	ptr2line := make(map[string]int)

	d := jsontext.NewDecoder(bytes.NewReader(b))
	for {
		// Any error (io.EOF at end, or a malformed-input error that can't occur
		// on our own freshly-rendered bytes) ends the scan — best-effort.
		if _, err := d.ReadToken(); err != nil {
			break
		}
		p := string(d.StackPointer())
		if p == "" {
			continue
		}
		if _, seen := ptr2line[p]; seen {
			continue
		}
		line := lt.lineAt(d.InputOffset())
		ptr2line[p] = line
		line2ptr[line] = p
	}
	return newSpecIndex(line2ptr, ptr2line)
}

// BuildYAMLIndex builds a SpecIndex from YAML bytes by walking the yaml.Node
// tree, which carries a 1-based .Line per node. Pointers use the same RFC 6901
// escaping as the JSON side, so an index built from either render is
// interchangeable. Best-effort — a parse error yields an empty index.
func BuildYAMLIndex(b []byte) *SpecIndex {
	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		return newSpecIndex(map[int]string{}, map[string]int{})
	}
	line2ptr := make(map[int]string)
	ptr2line := make(map[string]int)
	// A document node wraps a single content node.
	for _, doc := range docContents(&root) {
		walkYAML(doc, "", line2ptr, ptr2line)
	}
	return newSpecIndex(line2ptr, ptr2line)
}

func docContents(n *yaml.Node) []*yaml.Node {
	if n.Kind == yaml.DocumentNode {
		return n.Content
	}
	return []*yaml.Node{n}
}

// walkYAML records a pointer→line for each mapping member and sequence element,
// recursing into nested mappings/sequences. Scalars are leaves (their line is
// recorded by the parent member/element).
func walkYAML(n *yaml.Node, prefix string, line2ptr map[int]string, ptr2line map[string]int) {
	switch n.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, val := n.Content[i], n.Content[i+1]
			ptr := prefix + "/" + escapePointer(key.Value)
			record(key.Line-1, ptr, line2ptr, ptr2line)
			walkYAML(val, ptr, line2ptr, ptr2line)
		}
	case yaml.SequenceNode:
		for i, val := range n.Content {
			ptr := prefix + "/" + strconv.Itoa(i)
			record(val.Line-1, ptr, line2ptr, ptr2line)
			walkYAML(val, ptr, line2ptr, ptr2line)
		}
	default:
		// Scalars and aliases are leaves; their line was recorded by the
		// enclosing member/element. DocumentNode is unwrapped in docContents.
	}
}

func record(line int, ptr string, line2ptr map[int]string, ptr2line map[string]int) {
	if _, seen := ptr2line[ptr]; seen {
		return
	}
	ptr2line[ptr] = line
	if _, taken := line2ptr[line]; !taken {
		line2ptr[line] = ptr
	}
}

// escapePointer applies RFC 6901 token escaping (~ → ~0, / → ~1), matching what
// jsontext.Pointer produces on the JSON side.
func escapePointer(tok string) string {
	tok = strings.ReplaceAll(tok, "~", "~0")
	tok = strings.ReplaceAll(tok, "/", "~1")
	return tok
}

// itoa is a tiny strconv.Itoa avoiding the import churn for one call site.
// lineTable maps byte offsets to 0-based line numbers via the positions of '\n'.
type lineTable struct {
	nlOffsets []int64 // sorted offsets of each '\n'
}

func newLineTable(b []byte) lineTable {
	var nl []int64
	for i, c := range b {
		if c == '\n' {
			nl = append(nl, int64(i))
		}
	}
	return lineTable{nlOffsets: nl}
}

// lineAt returns the 0-based line containing the byte at off (= number of
// newlines strictly before off).
func (lt lineTable) lineAt(off int64) int {
	return sort.Search(len(lt.nlOffsets), func(i int) bool { return lt.nlOffsets[i] >= off })
}
