// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// optionsDeliberatelyOmitted lists exported bools on codescan.Options that must
// NOT get an overlay row, with the reason. Anything not here and not in the
// overlay fails TestOptions_OverlayCoversEveryBoolKnob.
var optionsDeliberatelyOmitted = map[string]string{ //nolint:gochecknoglobals // table for the drift guard
	"DescWithRef": "deprecated in favour of EmitRefSiblings",
	"Debug":       "deprecated no-op; the stderr logger was retired",
}

func newOptionsModel(t *testing.T) *Model {
	t.Helper()
	m := New(codescan.Options{WorkDir: t.TempDir(), Packages: []string{"./..."}})
	t.Cleanup(m.Close)
	m.height = 40 // tall enough that nothing scrolls unless a test wants it to

	return m
}

// The overlay silently fell eleven knobs behind the v0.36 feature streak
// because nothing failed when they landed. This is what fails now.
func TestOptions_OverlayCoversEveryBoolKnob(t *testing.T) {
	m := newOptionsModel(t)

	covered := make(map[*bool]string, len(m.optToggles))
	for _, tg := range m.optToggles {
		covered[tg.ptr] = tg.label
	}

	cfg := reflect.ValueOf(&m.cfg).Elem()
	typ := cfg.Type()
	for i := range typ.NumField() {
		f := typ.Field(i)
		if !f.IsExported() || f.Type.Kind() != reflect.Bool {
			continue
		}

		ptr, ok := cfg.Field(i).Addr().Interface().(*bool)
		require.True(t, ok, "Options.%s", f.Name)

		if label, inOverlay := covered[ptr]; inOverlay {
			assert.Equal(t, f.Name, label,
				"the row for Options.%s should be labelled with the field name", f.Name)

			continue
		}
		if _, omitted := optionsDeliberatelyOmitted[f.Name]; omitted {
			continue
		}

		t.Errorf("codescan.Options.%s is a boolean knob with no row in the options "+
			"overlay. Add one (grouped), or add it to optionsDeliberatelyOmitted "+
			"with a reason.", f.Name)
	}
}

// The omission list must not rot either: an entry naming a field that no longer
// exists, or one that has since been given a row, is stale.
func TestOptions_OmissionListIsCurrent(t *testing.T) {
	m := newOptionsModel(t)

	labelled := make(map[string]bool, len(m.optToggles))
	for _, tg := range m.optToggles {
		labelled[tg.label] = true
	}

	typ := reflect.TypeOf(m.cfg)
	for name, reason := range optionsDeliberatelyOmitted {
		f, ok := typ.FieldByName(name)
		assert.True(t, ok, "optionsDeliberatelyOmitted names Options.%s, which no longer exists", name)
		assert.Equal(t, reflect.Bool, f.Type.Kind(), "Options.%s is not a bool", name)
		assert.False(t, labelled[name], "Options.%s is both omitted and in the overlay", name)
		assert.NotEmpty(t, reason, "Options.%s is omitted without a reason", name)
	}
}

// Every row must point into m.cfg — a row bound to a stray variable would
// toggle nothing, and the scan would ignore it.
func TestOptions_EveryRowPointsIntoTheConfig(t *testing.T) {
	m := newOptionsModel(t)

	inCfg := make(map[*bool]bool)
	cfg := reflect.ValueOf(&m.cfg).Elem()
	for i := range cfg.NumField() {
		if f := cfg.Type().Field(i); !f.IsExported() || f.Type.Kind() != reflect.Bool {
			continue
		}
		if ptr, ok := cfg.Field(i).Addr().Interface().(*bool); ok {
			inCfg[ptr] = true
		}
	}

	seen := make(map[string]bool, len(m.optToggles))
	for _, tg := range m.optToggles {
		assert.True(t, inCfg[tg.ptr], "row %q is not bound to a codescan.Options field", tg.label)
		assert.NotEmpty(t, tg.desc, "row %q has no description", tg.label)
		assert.NotEmpty(t, tg.group, "row %q has no group", tg.label)
		assert.False(t, seen[tg.label], "row %q appears twice", tg.label)
		seen[tg.label] = true
	}
}

// Rows are stored flat but rendered grouped, so each group must appear as one
// contiguous run — otherwise a header would be emitted twice.
func TestOptions_GroupsAreContiguous(t *testing.T) {
	m := newOptionsModel(t)

	var order []string
	started := make(map[string]bool)
	last := ""
	for _, tg := range m.optToggles {
		if tg.group == last {
			continue
		}
		assert.False(t, started[tg.group], "group %q is split across the list", tg.group)
		started[tg.group] = true
		order = append(order, tg.group)
		last = tg.group
	}

	assert.Len(t, order, optionGroupCount, "optionGroupCount must match the groups in use")
}

func TestOptions_DependentRowSaysWhyItIsInert(t *testing.T) {
	m := newOptionsModel(t)
	m.cfg.ScanModels = false

	view := stripANSI(m.optionsView())
	assert.Contains(t, view, "(needs ScanModels)",
		"PruneUnusedModels must say it does nothing without ScanModels")

	m.cfg.ScanModels = true
	assert.NotContains(t, stripANSI(m.optionsView()), "(needs ScanModels)",
		"...and stop saying so once the dependency holds")
}

// The inverse form: EmitXGoType is suppressed BY SkipExtensions rather than
// requiring it.
func TestOptions_InverseDependency(t *testing.T) {
	m := newOptionsModel(t)
	m.cfg.SkipExtensions = true

	view := stripANSI(m.optionsView())
	assert.Contains(t, view, "(moot: SkipExtensions)")

	m.cfg.SkipExtensions = false
	assert.NotContains(t, stripANSI(m.optionsView()), "(moot: SkipExtensions)")
}

func TestOptions_ViewShowsGroupsAndRows(t *testing.T) {
	m := newOptionsModel(t)

	view := stripANSI(m.optionsView())

	for _, g := range []string{groupScope, groupRefs, groupNaming, groupDocs, groupTypes} {
		assert.Contains(t, view, g, "group header")
	}
	for _, label := range []string{"ScanModels", "CleanGoDoc", "EmitXGoType", "AfterDeclComments"} {
		assert.Contains(t, view, label)
	}
}

// A short terminal must scroll rather than overflow, and the cursor must stay
// on screen wherever it is.
func TestOptions_ScrollsOnAShortTerminal(t *testing.T) {
	m := newOptionsModel(t)
	m.height = 16 // ~6 visible rows

	full, _ := m.optionsLines()
	require.Greater(t, len(full), m.optionsVisibleRows(), "precondition: the list overflows")

	// Walk to the last row; it must be visible at the end.
	for range len(m.optToggles) {
		_, _ = m.handleOptionsKey(tea.KeyMsg{Type: tea.KeyDown})
	}
	last := m.optToggles[len(m.optToggles)-1]
	assert.Contains(t, stripANSI(m.optionsView()), last.label,
		"the cursor row must be inside the scrolled window")

	// ...and the first row is no longer shown.
	assert.NotContains(t, stripANSI(m.optionsView()), m.optToggles[0].label)
}

func TestOptions_ToggleAndApply(t *testing.T) {
	m := newOptionsModel(t)
	m.optionsOpen = true
	require.False(t, m.cfg.ScanModels)

	_, _ = m.handleOptionsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	assert.True(t, m.cfg.ScanModels, "space toggles the row under the cursor")
	assert.True(t, m.optDirty)

	_, cmd := m.handleOptionsKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.optionsOpen)
	assert.NotNil(t, cmd, "a changed option triggers a rescan on close")
}

func TestOptions_CloseWithoutChangesDoesNotRescan(t *testing.T) {
	m := newOptionsModel(t)
	m.optionsOpen = true

	_, cmd := m.handleOptionsKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, m.optionsOpen)
	assert.Nil(t, cmd, "nothing changed, so nothing to re-run")
}

// Guards against a row whose description repeats its label, which reads as
// noise in a list this long.
func TestOptions_DescriptionsAddInformation(t *testing.T) {
	m := newOptionsModel(t)

	for _, tg := range m.optToggles {
		assert.NotEqual(t, strings.ToLower(tg.label), strings.ToLower(tg.desc),
			"row %q", tg.label)
		// The modal is as wide as its widest row, so descriptions are kept
		// terse rather than clipped at render time.
		assert.LessOrEqual(t, len(tg.desc), 40, "row %q description is too long for the modal", tg.label)
	}
}
