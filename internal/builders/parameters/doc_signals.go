// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parameters

import (
	"go/ast"
	"strings"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// fieldDocSignals carries the per-field doc-comment signals the
// parameter dispatcher reads upstream of the schema build: the
// `in:` location, presence of `swagger:ignore`, presence of
// `swagger:file`, and the `swagger:strfmt` argument when set.
// Replaces the four v1 regex helpers (parsers.ParamLocation /
// parsers.FileParam / parsers.StrfmtName / parsers.Ignored) with
// grammar lookups plus a small `in:` line scan.
type fieldDocSignals struct {
	in        string
	inSet     bool
	ignored   bool
	file      bool
	strfmt    string
	strfmtSet bool
}

// scanFieldDocSignals reads every signal the parameter dispatcher
// needs out of a pre-parsed block slice and the raw doc text.
// Callers should pass `p.ParseBlocks(afld.Doc)` so the
// common.Builder cache absorbs the parse cost.
//
// Returns the zero value when doc is nil.
//
// # Details
//
// See [§in-discriminator](./README.md#in-discriminator) — why `in:`
// is line-scanned rather than read as a grammar Property.
func scanFieldDocSignals(blocks []grammar.Block, doc *ast.CommentGroup) fieldDocSignals {
	var pd fieldDocSignals
	if doc == nil {
		return pd
	}

	for _, b := range blocks {
		switch b.AnnotationKind() { //nolint:exhaustive // only ignore/file/strfmt are relevant here
		case grammar.AnnIgnore:
			pd.ignored = true
		case grammar.AnnFile:
			pd.file = true
		case grammar.AnnStrfmt:
			if arg, ok := b.AnnotationArg(); ok && !strings.ContainsAny(arg, " \t") {
				pd.strfmt = arg
				pd.strfmtSet = true
			}
		}
	}

	if v, ok := scanInLocation(doc.Text()); ok {
		pd.in = v
		pd.inSet = true
	}

	return pd
}

// scanInLocation finds the first `in: X` (case-insensitive on `in`
// and on X) line in text and returns X canonicalised to the OAS v2
// closed vocabulary (`query` / `path` / `header` / `body` /
// `formData`) via [grammar.NormalizeIn]. Mirrors v1's `rxIn`
// semantics extended with Q29's case-insensitive value matching:
//
//	regexp: `[Ii]n\p{Zs}*:\p{Zs}*(query|path|header|body|formData)(?:\.)?$`
//	  + case-insensitive on the captured group (go-swagger codegen
//	    emits capitalised forms like `in: Body`).
//
// The `form` alias (Q27) is NOT accepted here — it is contained to
// the routes inline-param path in `internal/parsers/routebody`.
func scanInLocation(text string) (string, bool) {
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(line, "in:")
		if !ok {
			rest, ok = strings.CutPrefix(line, "In:")
		}
		if !ok {
			continue
		}
		v := strings.TrimSpace(rest)
		v = strings.TrimSuffix(v, ".")
		if canonical, ok := grammar.NormalizeIn(v, false); ok {
			return canonical, true
		}
	}
	return "", false
}

// strfmtFromDoc returns the argument of a `swagger:strfmt <name>`
// annotation present in blocks (the pre-parsed common.Builder cache
// slice for some CommentGroup). Single-word filter mirrors the
// schema package's `findAnnotationArg` rule.
func strfmtFromDoc(blocks []grammar.Block) (string, bool) {
	for _, b := range blocks {
		if b.AnnotationKind() != grammar.AnnStrfmt {
			continue
		}
		if arg, ok := b.AnnotationArg(); ok && !strings.ContainsAny(arg, " \t") {
			return arg, true
		}
	}
	return "", false
}
