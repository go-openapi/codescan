// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"path/filepath"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/index"
)

// Cross-ref outcome descriptions.
//
// A link can fail for genuinely different reasons, and conflating them sends the user hunting for a bug that isn't
// there: a node with no anchored ancestor was never produced from code, an InputSpec overlay node legitimately has no
// origin, and a node that resolved but isn't rendered is simply outside the active JSON/YAML view.
//
// All three are first-class answers rather than errors, which is why a resolution carries one whether or not it found
// anything.
const (
	noNodeDesc        = "(no node here)"
	noFileDesc        = "(no file open)"
	noAnchorDesc      = "no spec node anchored at or above this line"
	noProvenanceDesc  = "no provenance from the last scan"
	noSourceSuffix    = " · no source (not produced from code)"
	notRenderedSuffix = " · not rendered in this view"
)

// specTarget is where a coordinate resolves to in the rendered spec.
//
// Miss is empty when Found, and otherwise says which of the legitimate "nothing here" answers this is.
type specTarget struct {
	Pointer string
	Line    int
	Found   bool
	Miss    string
}

// sourceTarget is where a spec node resolves to in the Go source.
type sourceTarget struct {
	Pointer string
	Pos     token.Position
	Found   bool
	Miss    string
}

// missing builds an unresolved specTarget.
func missing(desc string) specTarget { return specTarget{Miss: desc} }

// resolveSourceToSpec finds the spec node produced by a source line (1-based).
//
// The nearest anchor at or above the line wins, which is what lets a click anywhere inside a type declaration find the
// definition it produced.
func resolveSourceToSpec(src *index.SourceIndex, spec *index.SpecIndex, file string, line int) specTarget {
	if file == "" {
		return missing(noFileDesc)
	}

	ptr, ok := src.PointerAt(file, line)
	if !ok {
		if src.Len() == 0 {
			return missing(noProvenanceDesc)
		}

		return missing(noAnchorDesc)
	}

	specLine, ok := spec.LineForPointer(ptr)
	if !ok {
		return missing(ptr + notRenderedSuffix)
	}

	return specTarget{Pointer: ptr, Line: specLine, Found: true}
}

// resolveFileToSpec finds the FIRST spec node a source file produced, for the one-shot locate from the tree.
func resolveFileToSpec(src *index.SourceIndex, spec *index.SpecIndex, path string) specTarget {
	ptr, ok := src.FirstAnchor(path)
	if !ok {
		return missing("no spec node produced by " + filepath.Base(path))
	}

	specLine, ok := spec.LineForPointer(ptr)
	if !ok {
		return missing("node not in the current spec view: " + ptr)
	}

	return specTarget{Pointer: ptr, Line: specLine, Found: true}
}

// resolveRefToSpec follows the $ref on a rendered line to the node it points at.
//
// Only local (`#/…`) refs are followable: the TUI renders one spec and is not a $ref resolver, so an external target
// is reported honestly rather than guessed at.
func resolveRefToSpec(refs *index.RefIndex, spec *index.SpecIndex, line int) specTarget {
	site, ok := refs.RefAt(line)
	if !ok {
		return missing("no $ref on this line")
	}
	if !site.Target.Local {
		return missing("external ref, not in this spec: " + site.Target.Raw)
	}

	specLine, ok := spec.LineForPointer(site.Target.Pointer)
	if !ok {
		return missing(site.Target.Pointer + notRenderedSuffix)
	}

	return specTarget{Pointer: site.Target.Pointer, Line: specLine, Found: true}
}

// resolveSpecToSource finds the source position behind the spec node on a rendered line.
//
// The two misses are told apart deliberately: no provenance at all means the last scan produced none, whereas a node
// that resolves but has no position of its own was not produced from code.
func resolveSpecToSource(spec *index.SpecIndex, src *index.SourceIndex, line int) sourceTarget {
	ptr, ok := spec.PointerAt(line)
	if !ok {
		return sourceTarget{Miss: noNodeDesc}
	}

	pos, ok := src.PositionFor(ptr)
	if !ok {
		if src.Len() == 0 {
			return sourceTarget{Pointer: ptr, Miss: ptr + " · " + noProvenanceDesc}
		}

		return sourceTarget{Pointer: ptr, Miss: ptr + noSourceSuffix}
	}

	return sourceTarget{Pointer: ptr, Pos: pos, Found: true}
}
