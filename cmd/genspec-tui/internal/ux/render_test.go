// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/validation"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The rest of the suite drives handleKey directly, which is the right altitude for testing what a
// key does - but it means the render path never runs. These call View and the lines it composes, so
// a panic or an empty pane in any state is caught here rather than by a user.

// view renders and strips the styling, leaving the text a user reads.
func view(t *testing.T, m *Model) string {
	t.Helper()

	return testutils.StripANSI(m.View())
}

func TestView(t *testing.T) {
	t.Run("before the first size message", func(t *testing.T) {
		m := testModel(t)
		m.ready = false

		assert.Equal(t, "loading…", m.View(),
			"nothing can be laid out until the terminal size is known")
	})

	t.Run("the three-pane layout", func(t *testing.T) {
		m := testModel(t, sized(100, 40), withSpecJSON(refSpecJSON))

		out := view(t, m)

		assert.Contains(t, out, "genspec-tui", "the header banner")
		assert.Contains(t, out, "source", "the tree pane title")
		assert.Contains(t, out, "definitions", "the spec pane body")
		assert.Contains(t, out, "tab/click: focus", "the status line")
	})

	t.Run("an open overlay covers everything", func(t *testing.T) {
		m := testModel(t, sized(100, 40), withSpecJSON(refSpecJSON))
		m.help.Open()

		out := view(t, m)

		assert.Contains(t, out, "Key bindings")
		assert.NotContains(t, out, "genspec-tui", "the header is behind the modal, not beside it")
	})

	t.Run("the options overlay wins over the help", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		m.help.Open()
		m.options.Open()

		out := view(t, m)

		assert.Contains(t, out, "Scanner options")
		assert.NotContains(t, out, "Key bindings")
	})
}

func TestStatusLine(t *testing.T) {
	t.Run("the default hint", func(t *testing.T) {
		m := testModel(t, sized(100, 40))

		assert.Contains(t, testutils.StripANSI(m.statusLine()), "o: options")
	})

	t.Run("viewing a file", func(t *testing.T) {
		m := testModel(t, sized(100, 40), viewing("user.go", "package p\n"), focusedOn(paneTree))

		assert.Contains(t, testutils.StripANSI(m.statusLine()), "viewing")
	})

	t.Run("editing a file", func(t *testing.T) {
		m := testModel(t, sized(100, 40), viewing("user.go", "package p\n"), focusedOn(paneTree))
		_ = m.fileView.StartEdit()

		assert.Contains(t, testutils.StripANSI(m.statusLine()), "editing")
	})

	t.Run("a selected diagnostic", func(t *testing.T) {
		m := testModel(t, sized(100, 40), focusedOn(paneDiag), withDiags(threeDiags()...))

		assert.Contains(t, testutils.StripANSI(m.statusLine()), "diagnostic 1/3")
	})

	// Both tabs are populated here on purpose: the counter has to come from the tab on SCREEN, not from whichever
	// list happens to be non-empty. Reporting the scan's cursor under a validation list counts a selection the user
	// cannot see, and against the wrong total.
	t.Run("a selected validation finding", func(t *testing.T) {
		m := testModel(t, sized(100, 40), focusedOn(paneDiag), withDiags(threeDiags()...))
		m.validation = ValidationState{
			Ran: true,
			Findings: []validation.Finding{
				{Severity: grammar.SeverityError, Message: "first"},
				{Severity: grammar.SeverityError, Message: "second"},
			},
			Cursor: 1,
		}
		m.diagTab = tabValidation

		out := testutils.StripANSI(m.statusLine())

		assert.Contains(t, out, "finding 2/2")
		assert.NotContains(t, out, "diagnostic", "the scan tab's counter must not leak into the validation tab")
	})

	t.Run("a notice outranks the pane hint", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		m.notice = "saved user.go"

		assert.Contains(t, testutils.StripANSI(m.statusLine()), "saved user.go")
	})

	t.Run("an active reference cycle", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		m.refs.Status = "ref 2/3 of /definitions/User"

		out := testutils.StripANSI(m.statusLine())
		assert.Contains(t, out, "REFS")
		assert.Contains(t, out, "ref 2/3")
	})

	t.Run("the search prompt takes the line", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		_ = m.search.Open()

		assert.Contains(t, testutils.StripANSI(m.statusLine()), "search spec",
			"the input's placeholder, so the prompt is what the user sees")
	})

	// The spec pane names the node under the cursor, and only advertises go-to-definition when
	// there is actually a $ref on that line.
	t.Run("a spec node", func(t *testing.T) {
		m := testModel(t, sized(100, 40), focusedOn(paneSpec), withSpecJSON(refSpecJSON))

		m.spec.SetCursor(refLine(t, "#/definitions/User"))
		onRef := testutils.StripANSI(m.statusLine())
		assert.Contains(t, onRef, "node /definitions/Team")
		assert.Contains(t, onRef, "enter: go to definition")

		m.spec.SetCursor(refLine(t, `"User": {`))
		offRef := testutils.StripANSI(m.statusLine())
		assert.Contains(t, offRef, "node /definitions/User")
		assert.NotContains(t, offRef, "enter: go to definition",
			"nothing to follow from here, so it is not advertised")
	})
}

func TestFollowBadge(t *testing.T) {
	for _, tc := range []struct {
		mode  followMode
		label string
	}{
		{followSpec, "SPEC ▸ SOURCE"},
		{followSource, "SOURCE ▸ SPEC"},
		{followDiag, "DIAG ▸ SOURCE + SPEC"},
	} {
		m := testModel(t, sized(100, 40))
		m.follow = tc.mode
		m.followTarget = "/definitions/User"

		out := testutils.StripANSI(m.statusLine())

		assert.Contains(t, out, tc.label)
		assert.Contains(t, out, "/definitions/User")
	}

	t.Run("before the driver has moved", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		m.follow = followSpec

		assert.Contains(t, testutils.StripANSI(m.statusLine()), "(move the cursor)")
	})
}

func TestHeaderLine(t *testing.T) {
	t.Run("stats and the ready state", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		m.scan.NumPaths, m.scan.NumDefs = 3, 7

		out := testutils.StripANSI(m.headerLine())

		assert.Contains(t, out, "3 paths · 7 defs")
		assert.Contains(t, out, "ready")
	})

	t.Run("while a scan runs", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		m.scan.Running = true

		assert.Contains(t, testutils.StripANSI(m.headerLine()), "scanning")
	})

	t.Run("the last scan's duration", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		m.scan.Elapsed = 1500 * time.Millisecond

		assert.Contains(t, testutils.StripANSI(m.headerLine()), "ready (2s)")
	})

	t.Run("a match count when a search is live", func(t *testing.T) {
		m := testModel(t, sized(100, 40), withSpecJSON(refSpecJSON))
		require.Positive(t, m.spec.Search("definitions"))

		assert.Contains(t, testutils.StripANSI(m.headerLine()), "match 1/")
	})

	// The banner is what reveals every other key, so a long work dir must never push it off the line.
	t.Run("a long work dir cannot crowd out the banner", func(t *testing.T) {
		m := testModel(t, sized(60, 40))
		m.cfg.WorkDir = strings.Repeat("/very-long-path-segment", 8)

		out := testutils.StripANSI(m.headerLine())

		assert.Contains(t, out, "h: help")
		assert.Contains(t, out, "…", "the work dir is trimmed from the left instead")
	})

	// The work dir used to be given a fixed allowance and every other field a hand-tuned constant to fit inside. The
	// stats widen with the spec, the tail gains a duration, the match counter appears mid-search - so past that constant
	// the line simply ran off the right of the screen.
	//
	// Driven at several widths with every field at its longest, since the failure is a function of their sum.
	t.Run("the whole line fits, whatever it is reporting", func(t *testing.T) {
		const deepPath = "/home/fred/src/github.com/go-openapi/codescan/.worktrees/feat/" +
			"tui-ux-enhancements/fixtures/enhancements/annotation-noise"

		for _, w := range []int{40, 60, 80, 120, 160, 200} {
			m := testModel(t, sized(w, 40), withSpecJSON(refSpecJSON))
			m.cfg.WorkDir = deepPath
			m.scan.NumPaths, m.scan.NumDefs = 1234, 5678
			m.scan.Elapsed = 63 * time.Second
			require.Positive(t, m.spec.Search("definitions"))

			assert.LessOrEqual(t, lipgloss.Width(m.headerLine()), w, "the header overflows a %d-column terminal", w)
		}
	})

	// Which end gives way matters: what went missing was the half reporting whether the scan had finished at all, which
	// is worth more columns than the directory it ran in.
	t.Run("a long work dir yields to the fields on its right", func(t *testing.T) {
		m := testModel(t, sized(120, 40))
		m.cfg.WorkDir = strings.Repeat("/very-long-path-segment", 8)
		m.scan.NumPaths, m.scan.NumDefs = 12, 34
		m.scan.Elapsed = 2 * time.Second

		out := testutils.StripANSI(m.headerLine())

		assert.Contains(t, out, "12 paths · 34 defs")
		assert.Contains(t, out, "ready (2s)")
	})
}

// TestStatusLineFits covers the other unbounded line.
//
// Several of its variants embed a JSON pointer, a follow target or a file path, none of which have a length limit.
func TestStatusLineFits(t *testing.T) {
	for _, w := range []int{40, 80, 120} {
		m := testModel(t, sized(w, 40), withSpecJSON(refSpecJSON), focusedOn(paneSpec))
		m.spec.SetCursor(refLine(t, `"User": {`))
		assert.LessOrEqual(t, lipgloss.Width(m.statusLine()), w, "the status line overflows a %d-column terminal", w)

		m.follow = followDiag
		m.followTarget = strings.Repeat("/definitions/VeryLongDefinitionName", 4)
		assert.LessOrEqual(t, lipgloss.Width(m.statusLine()), w, "the follow badge overflows a %d-column terminal", w)
	}
}

func TestHumanDuration(t *testing.T) {
	for _, tc := range []struct {
		in   time.Duration
		want string
	}{
		{947 * time.Millisecond, "947ms"},
		{0, "0ms"},
		{time.Second, "1s"},
		{3200 * time.Millisecond, "3s"},
		{59 * time.Second, "59s"},
		{time.Minute, "1m"},
		{2 * time.Minute, "2m"},
		{63 * time.Second, "1m 3s"},
	} {
		assert.Equal(t, tc.want, humanDuration(tc.in), "%s", tc.in)
	}
}

func TestShortenPath(t *testing.T) {
	assert.Equal(t, "/a/b", shortenPath("/a/b", 10), "a path that fits is untouched")
	assert.Equal(t, "…c/ddd", shortenPath("/aaa/bbb/ccc/ddd", 6))
	assert.Contains(t, shortenPath("/aaa/bbb/ccc/ddd", 0), "…",
		"an absurd budget is floored rather than panicking")
}

func TestLeftView(t *testing.T) {
	t.Run("browse mode shows the tree", func(t *testing.T) {
		m := testModel(t, sized(100, 40))

		assert.Contains(t, testutils.StripANSI(m.leftView(true)), "source")
	})

	t.Run("view mode shows the file", func(t *testing.T) {
		m := testModel(t, sized(100, 40), viewing("user.go", "package p\n"))

		assert.Contains(t, testutils.StripANSI(m.leftView(true)), "user.go")
	})

	// In spec- and diag-driven follow the source pane is the follower, so its nav line stays lit
	// even though the driving pane holds focus.
	t.Run("the follower keeps its nav line lit", func(t *testing.T) {
		m := testModel(t, sized(100, 40), viewing("user.go", "a\nb\nc\n"))
		m.follow = followSpec

		assert.NotEqual(t, m.leftView(false), m.leftView(true),
			"focus still changes the frame")
		assert.NotEmpty(t, testutils.StripANSI(m.leftView(false)))
	})
}

func TestFocusedContent(t *testing.T) {
	m := testModel(t, sized(100, 40), withSpecJSON(refSpecJSON), withDiags(threeDiags()...))

	m.focused = paneSpec
	assert.Contains(t, m.focusedContent(), "definitions")
	assert.NotNil(t, m.copyFocused(), "there is something to copy")

	m.focused = paneDiag
	assert.NotEmpty(t, m.focusedContent())

	m.focused = paneTree
	assert.NotEmpty(t, m.focusedContent(), "the tree renders its rows as text")

	m.leftMode = modeView
	m.fileView.SetFile("user.go", "package p\n")
	assert.Contains(t, m.focusedContent(), "package p")
}

// Nothing to copy must not enqueue a clipboard command.
//
// The command shells out, so an empty one would be a pointless subprocess.
//
// An empty open file is the reachable case: the spec pane always holds at least its placeholder,
// and the tree always holds its root row.
func TestCopyFocused_NothingToCopy(t *testing.T) {
	m := testModel(t, sized(100, 40), viewing("empty.go", ""), focusedOn(paneTree))

	require.Empty(t, m.focusedContent())
	assert.Nil(t, m.copyFocused())
}

// threeDiags is a short diagnostics list for the status line and the diagnostics pane.
func threeDiags() []grammar.Diagnostic {
	return []grammar.Diagnostic{
		grammar.Warnf(pos("a.go", 1, 1), grammar.CodeInvalidNumber, "one"),
		grammar.Warnf(pos("a.go", 2, 1), grammar.CodeInvalidNumber, "two"),
		grammar.Warnf(pos("b.go", 3, 1), grammar.CodeInvalidNumber, "three"),
	}
}

func TestInit(t *testing.T) {
	m := testModel(t, sized(100, 40))

	cmd := m.Init()

	require.NotNil(t, cmd, "the initial scan is always kicked off")
	assert.True(t, m.watch.Listening(), "a watcher over a real temp dir comes up")
	assert.NotNil(t, m.watch.Events())
}
