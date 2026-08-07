// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/validation"
)

// diagTab is which view the diagnostics pane is showing.
//
// The validation tab EXISTS only once a validation has run: an empty tab sitting there permanently would advertise a
// mode nobody asked for, and there is nothing truthful to put in it before `v` is pressed.
type diagTab int

const (
	tabScan diagTab = iota
	tabValidation
)

// ValidationState is the last validation's outcome.
//
// Held apart from ScanState because it has a different lifetime: a scan happens on its own, whereas a validation only
// happens when asked for — and is retired the moment the spec it judged is replaced.
type ValidationState struct {
	Findings []validation.Finding
	Err      error
	Ran      bool
	Cursor   int
}

// validationMsg carries a finished validation back to the update loop.
type validationMsg struct {
	findings []validation.Finding
	err      error
}

// startValidation validates the rendered spec off the event loop.
//
// It judges the JSON body as rendered rather than the *spec.Swagger behind it, so what is reported is what is on
// screen — and what a consumer would actually be handed.
func (m *Model) startValidation() tea.Cmd {
	body := m.scan.JSON
	if body == "" {
		return m.notify("(nothing to validate yet)")
	}

	return func() tea.Msg {
		findings, err := validation.Run([]byte(body))

		return validationMsg{findings: findings, err: err}
	}
}

// absorbValidation records a finished validation and shows it.
func (m *Model) absorbValidation(msg validationMsg) tea.Cmd {
	m.validation = ValidationState{Findings: msg.findings, Err: msg.err, Ran: true}
	m.diagTab = tabValidation
	m.focused = paneDiag
	m.refreshDiagnostics()

	if msg.err != nil {
		return m.notify("validation failed: %s", msg.err)
	}

	errs, warns := validation.Tally(msg.findings)
	if len(msg.findings) == 0 {
		return m.notify("the generated spec is valid")
	}

	return m.notify("%d validation errors, %d warnings", errs, warns)
}

// retireValidation drops the last validation and returns the pane to the scan tab.
//
// Called when a rescan lands. The findings judged the PREVIOUS document, and a list of complaints about a spec that no
// longer exists is worse than no list: every line of it invites navigating to a node that may have moved or gone.
// Pressing `v` again is a keystroke; a stale verdict silently believed is a bug hunt.
func (m *Model) retireValidation() {
	m.validation = ValidationState{}
	m.diagTab = tabScan
}

// toggleDiagTab switches the diagnostics pane between the scan and validation views.
//
// A no-op while nothing has been validated, since the tab it would switch to does not exist yet.
func (m *Model) toggleDiagTab() {
	if !m.validation.Ran {
		return
	}
	m.exitFollow() // the follower was mirroring the other tab's selection

	if m.diagTab == tabScan {
		m.diagTab = tabValidation
	} else {
		m.diagTab = tabScan
	}
	m.refreshDiagnostics()
}

// diagTabTitle renders the pane title, carrying the tab strip once there is more than one tab.
func (m *Model) diagTabTitle() string {
	if !m.validation.Ran {
		return "diagnostics"
	}

	scan, val := "scan", "validation"
	if m.diagTab == tabScan {
		scan = "[" + scan + "]"
	} else {
		val = "[" + val + "]"
	}

	return fmt.Sprintf("diagnostics · %s %s", scan, val)
}

// moveValidationCursor moves the validation selection by delta (clamped) and re-renders.
func (m *Model) moveValidationCursor(delta int) {
	if len(m.validation.Findings) == 0 {
		return
	}
	m.validation.Cursor = min(max(m.validation.Cursor+delta, 0), len(m.validation.Findings)-1)
	m.refreshDiagnostics()
}

// jumpValidationToSpec moves the spec pane to the selected finding's node and focuses it.
//
// The counterpart to the scan tab's Enter, and the reason this tab tracks only one pane: a validation finding is about
// the DOCUMENT. It names a JSON pointer and knows nothing about the Go source that produced it.
func (m *Model) jumpValidationToSpec() tea.Cmd {
	target, ok := m.validationTarget()
	if !ok {
		return m.notify("%s", target)
	}

	m.focused = paneSpec

	return m.notify("→ %s", target)
}

// driveValidationToSpec mirrors the spec pane to the selected finding WITHOUT moving focus.
func (m *Model) driveValidationToSpec() string {
	target, _ := m.validationTarget()

	return target
}

// validationTarget moves the spec cursor to the selected finding's node, reporting where it landed — or why it could
// not.
//
// Resolution walks UP the pointer to the nearest ancestor that is actually rendered, the same fallback the rescan
// cursor restore uses. That is what makes the validator's ambiguous dotted paths usable: a mis-split path costs
// precision, landing on the enclosing node, instead of failing outright.
func (m *Model) validationTarget() (string, bool) {
	if len(m.validation.Findings) == 0 {
		return "(no findings)", false
	}

	f := m.validation.Findings[m.validation.Cursor]
	if f.Pointer == "" || m.specIndex == nil {
		return "(this finding names no location in the spec)", false
	}

	for ptr := f.Pointer; ptr != ""; {
		if line, ok := m.specIndex.LineForPointer(ptr); ok {
			m.spec.JumpTo(line)

			return ptr, true
		}
		i := lastSlash(ptr)
		if i < 0 {
			break
		}
		ptr = ptr[:i]
	}

	return f.Pointer + notRenderedSuffix, false
}

// lastSlash is the index of the final pointer separator, or -1.
func lastSlash(ptr string) int {
	for i := len(ptr) - 1; i >= 0; i-- {
		if ptr[i] == '/' {
			return i
		}
	}

	return -1
}
