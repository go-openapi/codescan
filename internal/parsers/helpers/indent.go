// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"regexp"
	"strings"
)

// rxIndent matches leading whitespace/comment noise up to (and
// including) the first non-whitespace character, used to detect
// the common indent on the first line of a YAML body.
var rxIndent = regexp.MustCompile(`[\p{Zs}\t]*/*[\p{Zs}\t]*[^\p{Zs}\t]`)

// rxNotIndent matches the first non-whitespace character — used to
// cap the tab→space conversion so we only rewrite leading-indent
// tabs, not tabs embedded inside content.
var rxNotIndent = regexp.MustCompile(`[^\p{Zs}\t]`)

// RemoveIndent normalises the common leading indentation on a YAML
// body: it strips the first line's indent from every line and
// converts remaining tab indentation to two-space equivalents. The
// operations bridge calls this on grammar-isolated YAML fence
// bodies so tab-indented godoc-style YAML (e.g., the go119
// fixture) parses correctly.
func RemoveIndent(spec []string) []string {
	if len(spec) == 0 {
		return spec
	}

	loc := rxIndent.FindStringIndex(spec[0])
	if len(loc) < 2 || loc[1] <= 1 {
		return spec
	}

	s := make([]string, len(spec))
	copy(s, spec)

	for i := range s {
		if len(s[i]) < loc[1] {
			continue
		}

		s[i] = spec[i][loc[1]-1:]
		start := rxNotIndent.FindStringIndex(s[i])
		if len(start) < 2 || start[1] == 0 {
			continue
		}

		s[i] = strings.Replace(s[i], "\t", "  ", start[1])
	}

	return s
}
