// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliopts

import "strings"

// DefaultPatterns is the pattern a command scans when the caller names nothing.
const DefaultPatterns = "./..."

// SplitList parses a comma-separated flag into trimmed, non-empty entries.
//
// It returns nil when there is nothing usable - nil being what the scanner reads as "no filter".
func SplitList(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return out
}

// Patterns is the packages to scan: what the caller named, or everything under the working
// directory.
//
// Positional rather than a flag, so that `genspec ./api/...` reads the way every other Go command
// does - which is also why it cannot be a table entry.
func Patterns(args []string) []string {
	if len(args) > 0 {
		return args
	}

	return []string{DefaultPatterns}
}
