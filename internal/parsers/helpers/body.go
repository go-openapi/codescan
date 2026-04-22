// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package helpers is a grab-bag of small utilities shared by the
// grammar bridge builders (meta / operations / routes / schema /
// parameters / responses). Nothing in this package parses
// comments directly — it operates on pre-extracted line slices,
// raw values, or grammar.Block.ProseLines() output. Anything that
// reads comment groups lives in the scanner's classification
// layer or in internal/parsers/grammar/.
//
// Sub-parsers for richer body shapes (internal/parsers/yaml/,
// internal/parsers/enum/) are siblings to this package, not
// children — their size justifies a dedicated package per the
// sub-parser pattern.
package helpers

import (
	"fmt"
	"regexp"
	"strings"

	yamlparser "github.com/go-openapi/codescan/internal/parsers/yaml"
)

// rxLineLeader matches the leading comment noise the legacy
// multilineYAMLListParser stripped (rxUncommentNoDash equivalent):
// whitespace / tabs / slashes / asterisks, then an optional pipe.
// Preserves `-` so YAML list markers survive.
var rxLineLeader = regexp.MustCompile(`^[\p{Zs}\t/\*]*\|?`)

// YAMLListBody parses a meta/route block body as a strict YAML list
// and returns its stringified items. Mirrors the Q4 contract for
// `consumes:` / `produces:` bodies: leading comment/indent noise is
// stripped from each line (preserving `-` list markers), blank
// lines are dropped; a non-list body (scalar, map, parse error) is
// silently dropped — legacy code emits a WARNING log. Empty bodies
// return nil.
func YAMLListBody(body []string) []string {
	cleaned := make([]string, 0, len(body))
	for _, line := range body {
		stripped := rxLineLeader.ReplaceAllString(line, "")
		if strings.TrimSpace(stripped) == "" {
			continue
		}
		cleaned = append(cleaned, stripped)
	}
	if len(cleaned) == 0 {
		return nil
	}
	parsed, err := yamlparser.Parse(strings.Join(cleaned, "\n"))
	if err != nil {
		return nil
	}
	list, ok := parsed.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, fmt.Sprintf("%v", item))
	}
	return out
}

// SecurityRequirements parses a Security: block body. Each line is
// `name: scope1, scope2` (empty scope list permitted when the colon
// has no suffix). Values are comma-split, trimmed, and any
// secondary whitespace inside a scope truncates to the first word
// — matching v1's SetSecurity.Parse / NewSetSecurityScheme
// semantics (via strings.SplitAfter(" ")[0]).
func SecurityRequirements(body []string) []map[string][]string {
	const kvParts = 2
	var result []map[string][]string
	for _, raw := range body {
		kv := strings.SplitN(raw, ":", kvParts)
		if len(kv) < kvParts {
			continue
		}
		key := strings.TrimSpace(kv[0])
		scopes := []string{}
		for scope := range strings.SplitSeq(kv[1], ",") {
			tr := strings.TrimSpace(scope)
			if tr == "" {
				continue
			}
			// V1 quirk: a scope containing whitespace is truncated
			// at the first word. Preserved for parity; safe on the
			// single-word scopes every fixture uses today.
			tr = strings.SplitAfter(tr, " ")[0]
			scopes = append(scopes, strings.TrimSpace(tr))
		}
		result = append(result, map[string][]string{key: scopes})
	}
	return result
}

// SchemesList parses a `Schemes:` value — comma-split, trim each
// entry, drop empties. Returns nil when the input parses to zero
// entries.
func SchemesList(value string) []string {
	out := make([]string, 0)
	for s := range strings.SplitSeq(value, ",") {
		if ts := strings.TrimSpace(s); ts != "" {
			out = append(out, ts)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// DropEmpty filters out whitespace-only entries from a line slice.
// Used by the meta and routes bridges before handing body lines to
// YAML or extension parsers that choke on blank separators.
func DropEmpty(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
