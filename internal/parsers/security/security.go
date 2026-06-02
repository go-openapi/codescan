// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package security is a thin sub-parser for the `Security:` block
// body that appears under both `swagger:meta` and `swagger:route`.
// The body shape is one requirement per line:
//
//	name: scope1, scope2
//
// where the scope list may be empty (a bare `name:` is permitted)
// and scopes are whitespace-trimmed.
//
// Sibling of `internal/parsers/yaml/`: imported by
// `internal/parsers/grammar/` from the lexer-time dispatch in
// emitRawBlock (the same seam yaml.TypedExtensions plugs into for
// the `extensions:` keyword). Builders read the typed result via
// `grammar.Block.SecurityRequirements()` rather than re-parsing the
// raw body — same shape extensions already follows.
package security

import "strings"

// Requirement is one Security: line's contribution to the spec:
// a single-entry map from name → scope list. Multiple Requirements
// build up across lines.
type Requirement = map[string][]string

// Parse splits body on newlines and parses each non-blank line as a
// `name: scope1, scope2` security requirement. Empty body returns
// nil.
//
// V1 quirk preserved: a scope that contains whitespace truncates at
// its first word — fixtures today only use single-word scopes, so
// the truncation is invisible, but the regression risk is real
// enough to keep the behaviour locked.
func Parse(body string) []Requirement {
	if body == "" {
		return nil
	}
	return parseLines(strings.Split(body, "\n"))
}

func parseLines(lines []string) []Requirement {
	const kvParts = 2
	var result []Requirement
	for _, raw := range lines {
		kv := strings.SplitN(raw, ":", kvParts)
		if len(kv) < kvParts {
			continue
		}
		name := strings.TrimSpace(kv[0])
		if name == "" {
			continue
		}
		scopes := []string{}
		for scope := range strings.SplitSeq(kv[1], ",") {
			tr := strings.TrimSpace(scope)
			if tr == "" {
				continue
			}
			// V1 quirk: scope truncates at first whitespace.
			tr = strings.SplitAfter(tr, " ")[0]
			scopes = append(scopes, strings.TrimSpace(tr))
		}
		result = append(result, Requirement{name: scopes})
	}
	return result
}
