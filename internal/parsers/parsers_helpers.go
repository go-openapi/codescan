// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"regexp"
	"strings"
)

// cleanupScannerLines strips comment-marker noise (matching ur) from
// each line and trims leading/trailing all-empty runs. Used by the
// legacy body parsers for consumes/produces (rxUncommentNoDash) and
// extensions (rxUncommentHeaders), plus the grammar-side prose
// splitter in CollectScannerTitleDescription.
func cleanupScannerLines(lines []string, ur *regexp.Regexp) []string {
	if len(lines) == 0 {
		return lines
	}

	seenLine := -1
	var lastContent int

	uncommented := make([]string, 0, len(lines))
	for i, v := range lines {
		str := ur.ReplaceAllString(v, "")
		uncommented = append(uncommented, str)
		if str != "" {
			if seenLine < 0 {
				seenLine = i
			}
			lastContent = i
		}
	}

	if seenLine == -1 {
		return nil
	}

	return uncommented[seenLine : lastContent+1]
}

// CollectScannerTitleDescription splits header lines (free-form prose
// appearing before the first recognized tag in a comment block) into
// title and description slices, following the legacy SectionedParser
// heuristics: a blank-line separator splits after cleanup; absent
// that, a first line ending in punctuation or matching a markdown
// heading prefix is promoted to title; otherwise everything is
// description.
//
// Exposed for grammar-side bridges that reuse the same split over
// grammar.Block.ProseLines().
func CollectScannerTitleDescription(headers []string) (title, desc []string) {
	return collectScannerTitleDescription(headers)
}

// a shared function that can be used to split given headers
// into a title and description.
func collectScannerTitleDescription(headers []string) (title, desc []string) {
	hdrs := cleanupScannerLines(headers, rxUncommentHeaders)

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
		} else {
			desc = nil
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
