// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scanner"
)

// envelope is the machine-readable result: the document, everything the scanner observed about the
// source that produced it, and where each anchored spec node came from.
//
// The command's own shape - a document on stdout, diagnostics as prose on stderr - suits a person
// or a shell pipeline, and -format=spec keeps it. It is not enough for a program: prose
// carries positions only in the sense that they are printed in it, and provenance has nowhere to go
// at all. Rather than have every consumer grow a parser for our stderr, say it once in JSON.
//
// Nothing here is derived that a consumer could compute itself: definition and path counts come from
// the document, and elapsed time is the caller's own wall clock.
//
// Experimental, inheriting the status of [codescan.Options.OnProvenance]: the cross-ref surface may
// change while LSP / TUI integration matures.
type envelope struct {
	Spec        any          `json:"spec"`
	Diagnostics []jsonDiag   `json:"diagnostics"`
	Provenance  []jsonAnchor `json:"provenance"`
	Runtime     *jsonRuntime `json:"runtime,omitempty"`
}

// jsonRuntime holds what the scan cost to run, read from the Go runtime just after it finished.
//
// Carried, where the definition and path counts deliberately are not, because it cannot be derived
// from anything else here: it is a measurement of the process, and once the process has exited
// nobody can recover it. A host watching a WebAssembly guest sees the linear memory grow and cannot
// see what inside the guest asked for it.
type jsonRuntime struct {
	// Sys is memory obtained from the host. Under wasm the linear memory never shrinks, so this is
	// effectively the high-water mark rather than a reading at one moment.
	Sys uint64 `json:"sys"`
	// HeapAlloc counts what is still live now the scan is done - the spec, the index, the type graph.
	HeapAlloc uint64 `json:"heapAlloc"`
	// TotalAlloc is everything ever allocated, garbage included. The ratio against Sys says whether a
	// scan is holding memory or churning through it.
	TotalAlloc  uint64 `json:"totalAlloc"`
	Collections uint32 `json:"collections"`
}

// measure reads the runtime's own account of the scan.
//
// Deliberately without a runtime.GC() first: forcing a collection would report a tidier heap than the
// one the scan actually ran with, which is the opposite of the question being asked.
func measure() *jsonRuntime {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &jsonRuntime{
		Sys:         m.Sys,
		HeapAlloc:   m.HeapAlloc,
		TotalAlloc:  m.TotalAlloc,
		Collections: m.NumGC,
	}
}

// jsonDiag is one scanner observation, with its position split out so a consumer does not have to
// parse "file:line:col" back apart.
type jsonDiag struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Col      int    `json:"col,omitempty"`
}

// jsonAnchor ties an RFC 6901 pointer into the emitted document to the source that produced it.
//
// Only anchor nodes appear - type declarations, fields, values, route and meta blocks. A finer node
// resolves to its nearest anchored ancestor at the consumer, which is the contract the TUI's
// cross-ref already works to.
type jsonAnchor struct {
	Pointer string `json:"pointer"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Col     int    `json:"col,omitempty"`
}

// collector accumulates what the scan reports, and rewrites positions to be relative to the module.
//
// A caller holds the tree it handed over - "models/pet.go" - while the scan reports where it read it
// from, which under a WASI guest is a mount point the caller chose and under a native run is an
// absolute path. Relativising here rather than at each consumer keeps that mapping in one place.
type collector struct {
	root  string
	diags []jsonDiag
	nodes []jsonAnchor
}

func (c *collector) onDiagnostic(d codescan.Diagnostic) {
	file, line, col := c.at(d.Pos)
	c.diags = append(c.diags, jsonDiag{
		Severity: d.Severity.String(),
		Code:     string(d.Code),
		Message:  d.Message,
		File:     file,
		Line:     line,
		Col:      col,
	})
}

func (c *collector) onProvenance(p scanner.Provenance) {
	file, line, col := c.at(p.Pos)
	c.nodes = append(c.nodes, jsonAnchor{Pointer: p.Pointer, File: file, Line: line, Col: col})
}

// result assembles the envelope.
//
// Diagnostics keep the order they were reported in, which is the order a reader would have seen them
// on stderr. Anchors are sorted by pointer: they are a lookup table, so emission order says nothing,
// and sorting makes the same scan produce the same bytes.
func (c *collector) result(doc any) envelope {
	nodes := c.nodes
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].Pointer < nodes[j].Pointer })

	// Never null: a consumer should be able to range over these without checking.
	diags := c.diags
	if diags == nil {
		diags = []jsonDiag{}
	}
	if nodes == nil {
		nodes = []jsonAnchor{}
	}

	return envelope{Spec: doc, Diagnostics: diags, Provenance: nodes, Runtime: measure()}
}

// at splits a position, expressing the file relative to the module root where it can.
//
// A position outside the root - a dependency read from the module cache, or from GOROOT - is left
// absolute rather than turned into a chain of "..", which names it no better and reads worse. A
// consumer distinguishes the two by whether the path matches a file it holds.
func (c *collector) at(pos token.Position) (string, int, int) {
	if !pos.IsValid() {
		return "", 0, 0
	}

	return c.relative(pos.Filename), pos.Line, pos.Column
}

func (c *collector) relative(file string) string {
	if file == "" || c.root == "" || !filepath.IsAbs(file) {
		return filepath.ToSlash(file)
	}

	rel, err := filepath.Rel(c.root, file)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(file)
	}

	return filepath.ToSlash(rel)
}
