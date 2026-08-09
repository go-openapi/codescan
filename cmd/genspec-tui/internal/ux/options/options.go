// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/key"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// Option groups, in the order the overlay shows them.
//
// Seventeen flat rows is a wall.
// These are the same divisions the docs already uses to describe the knobs, so the overlay and the docs agree on where
// a setting lives.
const (
	groupScope   = "discovery & scope"
	groupRefs    = "$ref & composition"
	groupNaming  = "naming"
	groupDocs    = "docs & comments"
	groupTypes   = "types & extensions"
	groupLoading = "package loading"

	// groupCount is how many of the above appear, for sizing only.
	groupCount = 6
)

// dep records that a toggle only bites when another one holds a particular value.
//
// Several codescan knobs are modifiers rather than independent switches - PruneUnusedModels does nothing without
// ScanModels, EmitXGoType does nothing while SkipExtensions suppresses every x-go-* extension - and an overlay that
// let you tick them without saying so would quietly lie about what it did.
type dep struct {
	ptr   *bool
	on    bool   // the value ptr must hold for the dependent toggle to matter
	label string // the dependency's name, for the explanatory suffix
}

// satisfied reports whether the dependency currently holds.
func (d *dep) satisfied() bool { return d == nil || *d.ptr == d.on }

// note renders the "...but only when X" suffix shown while the dependency is unmet.
func (d *dep) note() string {
	if d.on {
		return " (needs " + d.label + ")"
	}

	return " (moot: " + d.label + ")"
}

// toggle binds a row to a boolean field of the scan config.
//
// It carries a short human description, the field names alone being cryptic, and the section it belongs to.
//
// Rows are stored flat and grouped at render time, so cursor movement stays a plain index and headers can never be
// landed on.
type toggle struct {
	group string
	label string
	desc  string
	ptr   *bool
	dep   *dep
}

// Overlay is the scanner-options modal.
//
// It holds a pointer to the live scan config: ticking a row writes straight through to the Options the next scan runs
// with, so there is no second copy to keep in step.
type Overlay struct {
	cfg *codescan.Options

	width, height int

	isOpen  bool
	cursor  int
	dirty   bool
	toggles []toggle
}

// New builds a closed options overlay over the given scan config.
//
// EVERY exported bool on codescan.Options belongs in the table below; TestOptions_OverlayCoversEveryBoolKnob fails
// when one is added without a row, which is how this list fell eleven knobs behind the v0.36 feature streak before
// anyone noticed.
func New(cfg *codescan.Options) Overlay {
	// Aliased because two rows depend on them.
	depScanModels, depSkipExtensions := &cfg.ScanModels, &cfg.SkipExtensions

	return Overlay{
		cfg: cfg,
		toggles: []toggle{
			// Discovery & scope.
			{groupScope, "ScanModels", "also emit swagger:model definitions", depScanModels, nil},
			{
				groupScope, "PruneUnusedModels", "drop models nothing references", &cfg.PruneUnusedModels,
				&dep{depScanModels, true, "ScanModels"},
			},
			{groupScope, "ExcludeDeps", "skip packages outside the module", &cfg.ExcludeDeps, nil},

			// $ref & composition.
			{groupRefs, "RefAliases", "$ref aliases instead of expanding", &cfg.RefAliases, nil},
			{groupRefs, "TransparentAliases", "aliases never become definitions", &cfg.TransparentAliases, nil},
			{groupRefs, "EmitRefSiblings", "description beside $ref, not allOf", &cfg.EmitRefSiblings, nil},
			{
				groupRefs, "SkipAllOfCompounding", "never wrap in allOf; drops validations",
				&cfg.SkipAllOfCompounding, nil,
			},
			{groupRefs, "DefaultAllOfForEmbeds", "plain embeds compose via allOf", &cfg.DefaultAllOfForEmbeds, nil},

			// Naming.
			{groupNaming, "EmitHierarchicalNames", "nest colliding names, not concats", &cfg.EmitHierarchicalNames, nil},
			{
				groupNaming, "SkipJSONifyInterfaceMethods", "interface methods keep their names",
				&cfg.SkipJSONifyInterfaceMethods, nil,
			},

			// Docs & comments.
			{
				groupDocs, "SingleLineCommentAsDescription", "one-line doc is description, not title",
				&cfg.SingleLineCommentAsDescription, nil,
			},
			{groupDocs, "AfterDeclComments", "annotations inside or after a decl", &cfg.AfterDeclComments, nil},
			{groupDocs, "CleanGoDoc", "strip godoc-only syntax from prose", &cfg.CleanGoDoc, nil},
			{groupDocs, "SkipEnumDescriptions", "enum names only on x-go-enum-desc", &cfg.SkipEnumDescriptions, nil},

			// Types & extensions.
			{groupTypes, "SetXNullableForPointers", "pointer fields get x-nullable", &cfg.SetXNullableForPointers, nil},
			{groupTypes, "SkipExtensions", "omit x-go-* vendor extensions", depSkipExtensions, nil},
			{
				groupTypes, "EmitXGoType", "stamp x-go-type on definitions", &cfg.EmitXGoType,
				&dep{depSkipExtensions, false, "SkipExtensions"},
			},

			// Package loading. Last because it is experimental and rarely touched: the cursor opens on the
			// row people actually came for.
			{
				groupLoading, "ToolchainFreeLoader", "load packages without the go command",
				&cfg.ToolchainFreeLoader, nil,
			},
			{
				groupLoading, "StubStdlib", "synthesize the stdlib, skip GOROOT",
				&cfg.StubStdlib, &dep{&cfg.ToolchainFreeLoader, true, "ToolchainFreeLoader"},
			},
			{
				groupLoading, "SkipCompiledDependencies", "read dep source, not the build cache",
				&cfg.SkipCompiledDependencies, &dep{&cfg.ToolchainFreeLoader, false, "ToolchainFreeLoader"},
			},
		},
	}
}

// SetSize fits the overlay to outer dimensions w×h (border + title reserved).
func (o *Overlay) SetSize(w, h int) {
	o.width = w
	o.height = h
}

// IsOpen reports whether the overlay is currently covering the UI.
func (o *Overlay) IsOpen() bool { return o.isOpen }

// Dirty reports whether a toggle was flipped since the overlay was last opened.
//
// It stays readable after Close, which is what lets the model apply the change on dismissal - until the model says it
// has, with MarkApplied.
func (o *Overlay) Dirty() bool { return o.dirty }

// MarkApplied acknowledges that the model has carried the change out.
//
// Without it the flag would survive its own application, and every later keystroke that reached ANY overlay would ask
// for the same rescan again.
func (o *Overlay) MarkApplied() { o.dirty = false }

// Open shows the overlay with a clean dirty flag.
//
// The cursor is deliberately left where it was: reopening to flip the neighbouring knob is the common case.
func (o *Overlay) Open() {
	o.isOpen = true
	o.dirty = false
}

// Close hides the overlay, leaving Dirty readable.
func (o *Overlay) Close() { o.isOpen = false }

// HandleKey drives the modal: move the cursor, toggle a boolean with space/enter, or dismiss.
//
// Dismissal only closes the overlay - the model asks Dirty afterwards and decides what applying means.
func (o *Overlay) HandleKey(msg tea.KeyMsg) tea.Cmd {
	last := len(o.toggles) - 1
	if delta, ok := key.Nav(key.MsgBinding(msg), o.visibleRows(), len(o.toggles)); ok {
		o.cursor = min(max(o.cursor+delta, 0), last)

		return nil
	}

	switch key.MsgBinding(msg) {
	case key.Space, key.Enter:
		t := o.toggles[o.cursor]
		*t.ptr = !*t.ptr
		o.dirty = true
	case key.Esc, key.O:
		o.Close()
	}

	return nil
}

// View renders the modal: a bordered list of boolean toggles with checkboxes and a cursor caret.
func (o *Overlay) View() string {
	all, cursorLine := o.lines()
	// Measured over EVERY row, before the window is taken: a frame sized to what is currently visible changes width as
	// the list scrolls under it, which reads as the box twitching rather than the content moving.
	width := o.contentWidth(all)
	lines := windowAround(all, cursorLine, o.visibleRows())

	var b strings.Builder
	b.WriteString(theme.Accent().Render(optionsTitle))
	fmt.Fprintf(&b, "  %s\n\n", theme.Status().Render(fmt.Sprintf("(%d)", len(o.toggles))))
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n\n")
	b.WriteString(theme.Status().Render(optionsFooter))

	return theme.ModalAt(width).Render(b.String())
}

// The modal's fixed texts, named so the width can be measured against them as well as against the rows.
const (
	optionsTitle  = "Scanner options"
	optionsFooter = "↑↓/jk: move · space: toggle · esc/o: apply & close"
)

// contentWidth is the width to pin the frame at, given every row the overlay can show.
//
// Deliberately not capped to the terminal: a cap makes lipgloss WRAP the rows that overflow, splitting a toggle's label
// from its explanation, whereas a modal wider than a narrow terminal is simply clipped - which is what it already did.
func (o *Overlay) contentWidth(all []string) int {
	return max(
		lipgloss.Width(strings.Join(all, "\n")),
		lipgloss.Width(fmt.Sprintf("%s  (%d)", optionsTitle, len(o.toggles))),
		lipgloss.Width(optionsFooter),
	)
}

// visibleRows is how many rendered rows fit between the modal's chrome (border, padding, title, footer).
//
// The list outgrew a fixed layout at seventeen knobs plus headers, so it scrolls rather than overflowing a short
// terminal.
func (o *Overlay) visibleRows() int {
	const chrome = 10 // border 2 + padding 2 + title 2 + footer 2, with slack

	return max(o.height-chrome, 3)
}

// lines renders the grouped rows and reports which rendered line the cursor sits on.
//
// Group headers are emitted as the group changes, so they are never navigable - the cursor indexes o.toggles, not
// these lines.
func (o *Overlay) lines() ([]string, int) {
	labelW := 0
	for _, t := range o.toggles {
		labelW = max(labelW, len(t.label))
	}

	lines := make([]string, 0, len(o.toggles)+2*groupCount)
	cursorLine, lastGroup := 0, ""
	for i, t := range o.toggles {
		if t.group != lastGroup {
			if lastGroup != "" {
				lines = append(lines, "")
			}
			lines = append(lines, theme.Accent().Render(t.group))
			lastGroup = t.group
		}
		if i == o.cursor {
			cursorLine = len(lines)
		}
		lines = append(lines, o.row(t, i == o.cursor, labelW))
	}

	return lines, cursorLine
}

// row renders one toggle.
//
// A row whose dependency is unmet is dimmed and says why: ticking it would otherwise look like it had done something.
func (o *Overlay) row(t toggle, selected bool, labelW int) string {
	box := "[ ]"
	if *t.ptr {
		box = "[x]"
	}
	caret := "  "
	if selected {
		caret = "▸ "
	}

	desc := t.desc
	inert := !t.dep.satisfied()
	if inert {
		desc += t.dep.note()
	}

	head := fmt.Sprintf("%s%s %-*s  ", caret, box, labelW, t.label)
	switch {
	case selected:
		// Highlight the whole row, description included.
		return theme.Selected().Render(head + desc)
	case inert:
		return theme.Status().Render(head + desc)
	default:
		return head + theme.Status().Render(desc)
	}
}

// windowAround returns at most size lines, scrolled to keep cursor visible.
//
// The cursor is kept as far from the edges as the list allows.
func windowAround(lines []string, cursor, size int) []string {
	if len(lines) <= size {
		return lines
	}
	top := min(max(cursor-size/2, 0), len(lines)-size)

	return lines[top : top+size]
}
