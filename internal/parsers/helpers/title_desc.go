// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"regexp"
	"strings"
)

// rxUncommentHeaders strips the leading `[whitespace/tabs/slashes/
// asterisks/dashes]*|?` prefix from a header line — used by
// CollectScannerTitleDescription to normalise comment-marker noise
// before the title/description split.
var rxUncommentHeaders = regexp.MustCompile(`^[\p{Zs}\t/\*-]*\|?`)

// rxPunctuationEnd matches a unicode punctuation character at
// end-of-line; a prose first line that ends with one is promoted
// to title when no blank separates title from description.
var rxPunctuationEnd = regexp.MustCompile(`\p{Po}$`)

// rxTitleStart matches a leading `# ` / `## ` markdown heading
// prefix — another trigger for the first-line-is-title heuristic.
var rxTitleStart = regexp.MustCompile(`^[#]+\p{Zs}+`)

// CollectScannerTitleDescription splits header lines (free-form
// prose appearing before the first recognized tag in a comment
// block) into title and description slices, following the legacy
// SectionedParser heuristics:
//
//   - A blank-line separator splits after cleanup.
//   - Absent that, a first line ending in punctuation or matching
//     a markdown heading prefix is promoted to title.
//   - Otherwise everything is description.
//
// Used by the grammar-side bridges (schema decl / operations /
// routes / meta) to reconstruct v1's title/description shapes.
func CollectScannerTitleDescription(headers []string) (title, desc []string) {
	hdrs := CleanupScannerLines(headers, rxUncommentHeaders)

	idx := -1
	for i, line := range hdrs {
		if strings.TrimSpace(line) == "" {
			idx = i
			break
		}
	}

	if idx > -1 {
		title = hdrs[:idx]
		if len(title) > 0 {
			title[0] = rxTitleStart.ReplaceAllString(title[0], "")
		}
		if len(hdrs) > idx+1 {
			desc = hdrs[idx+1:]
		}
		return title, desc
	}

	if len(hdrs) > 0 {
		line := hdrs[0]
		switch {
		case rxPunctuationEnd.MatchString(line):
			title = []string{line}
			desc = hdrs[1:]
		case rxTitleStart.MatchString(line):
			title = []string{rxTitleStart.ReplaceAllString(line, "")}
			desc = hdrs[1:]
		default:
			desc = hdrs
		}
	}

	return title, desc
}
