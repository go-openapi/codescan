// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"regexp"
	"strings"
)

// JoinDropLast joins lines with "\n" and, if the trailing line is
// whitespace-only, drops it first. Mirrors the legacy
// SectionedParser's description-accumulator shape so bridge
// outputs match v1 parity on field/method descriptions.
func JoinDropLast(lines []string) string {
	l := len(lines)
	lns := lines
	if l > 0 && len(strings.TrimSpace(lines[l-1])) == 0 {
		lns = lines[:l-1]
	}
	return strings.Join(lns, "\n")
}

// Setter returns a closure that joins lines and writes to target —
// the shape the SectionedParser title/description callbacks
// expected.
func Setter(target *string) func([]string) {
	return func(lines []string) {
		*target = JoinDropLast(lines)
	}
}

// CleanupScannerLines strips the regex's match from each line and
// trims leading / trailing runs of now-empty lines. Used by the
// legacy-body parsers (extensions) and by
// CollectScannerTitleDescription.
func CleanupScannerLines(lines []string, ur *regexp.Regexp) []string {
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
