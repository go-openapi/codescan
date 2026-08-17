// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"strings"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// renderSpans colours one raw line according to the lexical runs on it.
//
// The ORDER of operations is the whole point.
// Truncation happens on the raw text, at rune boundaries, before any escape exists; only then is each run wrapped in
// its style.
//
// Colour-then-truncate - what you are forced into when a highlighting library hands back a finished string - cuts
// through escape sequences and drops their resets, which is how a highlighted pane ends up bleeding colour across the
// rest of the screen.
//
// The fit is also what makes the line safe to hand to the styling layer, since it is where control characters are
// encoded. Nothing here may reach theme.Syntax on a path that skips it: raw text is the scanned repository's, and an
// ESC in it would be read by the terminal as a command rather than drawn.
//
// spans record only where each run starts, so a run extends to the next span's column. width <= 0 renders nothing; no
// spans renders the raw (fitted) text.
func renderSpans(raw string, spans []theme.Span, width int) string {
	if width <= 0 {
		return ""
	}
	if len(spans) == 0 {
		return fit(raw, width)
	}

	text := fit(raw, width) // raw, still - the ellipsis is plain by design
	runes := []rune(text)

	var b strings.Builder
	for i, sp := range spans {
		start := sp.Col - 1 // spans are 1-based
		if start >= len(runes) {
			break
		}
		if start < 0 {
			start = 0
		}

		end := len(runes)
		if i+1 < len(spans) {
			end = min(spans[i+1].Col-1, len(runes))
		}
		if end <= start {
			continue
		}

		// Anything before the first run (indentation) is emitted unstyled.
		if i == 0 && start > 0 {
			b.WriteString(string(runes[:start]))
		}
		b.WriteString(theme.Syntax(sp.Kind).Render(string(runes[start:end])))
	}

	// A trailing remainder can only appear if the line ends after the last run.
	if last := spans[len(spans)-1]; last.Col-1 < len(runes) {
		return b.String()
	}

	return string(runes)
}
