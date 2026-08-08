// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// deliberatelyOmitted lists exported bools on codescan.Options that must NOT get an overlay row, with the reason.
//
// Anything not here and not in the overlay fails TestOptions_OverlayCoversEveryBoolKnob.
var deliberatelyOmitted = map[string]string{ //nolint:gochecknoglobals // table for the drift guard
	"DescWithRef": "deprecated in favour of EmitRefSiblings",
	"Debug":       "deprecated no-op; the stderr logger was retired",
}

// newOverlay builds an overlay over its own config, tall enough that nothing scrolls unless a test wants it to.
func newOverlay(t *testing.T, height int) (*Overlay, *codescan.Options) {
	t.Helper()

	cfg := &codescan.Options{WorkDir: t.TempDir(), Packages: []string{"./..."}}
	o := New(cfg)
	o.SetSize(100, height)

	return &o, cfg
}

// The overlay silently fell eleven knobs behind the v0.36 feature streak because nothing failed when they landed.
//
// This is what fails now.
func TestOptions_OverlayCoversEveryBoolKnob(t *testing.T) {
	o, cfgPtr := newOverlay(t, 40)

	covered := make(map[*bool]string, len(o.toggles))
	for _, tg := range o.toggles {
		covered[tg.ptr] = tg.label
	}

	cfg := reflect.ValueOf(cfgPtr).Elem()
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
		if _, omitted := deliberatelyOmitted[f.Name]; omitted {
			continue
		}

		t.Errorf("codescan.Options.%s is a boolean knob with no row in the options "+
			"overlay. Add one (grouped), or add it to deliberatelyOmitted "+
			"with a reason.", f.Name)
	}
}

// The omission list must not rot either.
//
// An entry naming a field that no longer exists, or one that has since been given a row, is stale.
func TestOptions_OmissionListIsCurrent(t *testing.T) {
	o, cfg := newOverlay(t, 40)

	labelled := make(map[string]bool, len(o.toggles))
	for _, tg := range o.toggles {
		labelled[tg.label] = true
	}

	typ := reflect.TypeOf(*cfg)
	for name, reason := range deliberatelyOmitted {
		f, ok := typ.FieldByName(name)
		assert.True(t, ok, "deliberatelyOmitted names Options.%s, which no longer exists", name)
		assert.Equal(t, reflect.Bool, f.Type.Kind(), "Options.%s is not a bool", name)
		assert.False(t, labelled[name], "Options.%s is both omitted and in the overlay", name)
		assert.NotEmpty(t, reason, "Options.%s is omitted without a reason", name)
	}
}

// Every row must point into the config the overlay was built over.
//
// A row bound to a stray variable would toggle nothing, and the scan would ignore it.
func TestOptions_EveryRowPointsIntoTheConfig(t *testing.T) {
	o, cfgPtr := newOverlay(t, 40)

	inCfg := make(map[*bool]bool)
	cfg := reflect.ValueOf(cfgPtr).Elem()
	for i := range cfg.NumField() {
		if f := cfg.Type().Field(i); !f.IsExported() || f.Type.Kind() != reflect.Bool {
			continue
		}
		if ptr, ok := cfg.Field(i).Addr().Interface().(*bool); ok {
			inCfg[ptr] = true
		}
	}

	seen := make(map[string]bool, len(o.toggles))
	for _, tg := range o.toggles {
		assert.True(t, inCfg[tg.ptr], "row %q is not bound to a codescan.Options field", tg.label)
		assert.NotEmpty(t, tg.desc, "row %q has no description", tg.label)
		assert.NotEmpty(t, tg.group, "row %q has no group", tg.label)
		assert.False(t, seen[tg.label], "row %q appears twice", tg.label)
		seen[tg.label] = true
	}
}

// Rows are stored flat but rendered grouped, so each group must appear as one contiguous run.
//
// Otherwise a header would be emitted twice.
func TestOptions_GroupsAreContiguous(t *testing.T) {
	o, _ := newOverlay(t, 40)

	var order []string
	started := make(map[string]bool)
	last := ""
	for _, tg := range o.toggles {
		if tg.group == last {
			continue
		}
		assert.False(t, started[tg.group], "group %q is split across the list", tg.group)
		started[tg.group] = true
		order = append(order, tg.group)
		last = tg.group
	}

	assert.Len(t, order, groupCount, "groupCount must match the groups in use")
}

func TestOptions_DependentRowSaysWhyItIsInert(t *testing.T) {
	o, cfg := newOverlay(t, 40)
	cfg.ScanModels = false

	view := testutils.StripANSI(o.View())
	assert.Contains(t, view, "(needs ScanModels)",
		"PruneUnusedModels must say it does nothing without ScanModels")

	cfg.ScanModels = true
	assert.NotContains(t, testutils.StripANSI(o.View()), "(needs ScanModels)",
		"...and stop saying so once the dependency holds")
}

// The inverse form: EmitXGoType is suppressed BY SkipExtensions rather than requiring it.
func TestOptions_InverseDependency(t *testing.T) {
	o, cfg := newOverlay(t, 40)
	cfg.SkipExtensions = true

	view := testutils.StripANSI(o.View())
	assert.Contains(t, view, "(moot: SkipExtensions)")

	cfg.SkipExtensions = false
	assert.NotContains(t, testutils.StripANSI(o.View()), "(moot: SkipExtensions)")
}

func TestOptions_ViewShowsGroupsAndRows(t *testing.T) {
	o, _ := newOverlay(t, 40)

	view := testutils.StripANSI(o.View())

	for _, g := range []string{groupScope, groupRefs, groupNaming, groupDocs, groupTypes} {
		assert.Contains(t, view, g, "group header")
	}
	for _, label := range []string{"ScanModels", "CleanGoDoc", "EmitXGoType", "AfterDeclComments"} {
		assert.Contains(t, view, label)
	}
}

// A short terminal must scroll rather than overflow, and the cursor must stay on screen wherever it is.
func TestOptions_ScrollsOnAShortTerminal(t *testing.T) {
	o, _ := newOverlay(t, 16) // ~6 visible rows

	full, _ := o.lines()
	require.Greater(t, len(full), o.visibleRows(), "precondition: the list overflows")

	// Walk to the last row; it must be visible at the end.
	for range len(o.toggles) {
		_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	}
	last := o.toggles[len(o.toggles)-1]
	assert.Contains(t, testutils.StripANSI(o.View()), last.label,
		"the cursor row must be inside the scrolled window")

	// ...and the first row is no longer shown.
	assert.NotContains(t, testutils.StripANSI(o.View()), o.toggles[0].label)
}

func TestPaging_OptionsPopup(t *testing.T) {
	o, _ := newOverlay(t, 40)
	o.Open()
	last := len(o.toggles) - 1

	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, last, o.cursor)

	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyHome})
	assert.Zero(t, o.cursor)

	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Positive(t, o.cursor)
	assert.LessOrEqual(t, o.cursor, last, "paging never runs off the end")

	// Clamped rather than wrapping, at both ends.
	for range len(o.toggles) {
		_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyPgDown})
	}
	assert.Equal(t, last, o.cursor)
	for range len(o.toggles) {
		_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyPgUp})
	}
	assert.Zero(t, o.cursor)
}

func TestOptions_TogglingMarksItDirty(t *testing.T) {
	o, cfg := newOverlay(t, 40)
	o.Open()
	require.False(t, cfg.ScanModels)
	require.False(t, o.Dirty())

	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	assert.True(t, cfg.ScanModels, "space toggles the row under the cursor")
	assert.True(t, o.Dirty())

	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, o.IsOpen())
	assert.True(t, o.Dirty(), "the flag outlives the close — that is what the model applies on")
}

// Reopening starts a fresh round: whatever the last one changed has already been applied.
func TestOptions_ReopeningClearsDirty(t *testing.T) {
	o, _ := newOverlay(t, 40)
	o.Open()
	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	require.True(t, o.Dirty())

	o.Open()

	assert.False(t, o.Dirty())
}

func TestOptions_CloseWithoutChangesIsNotDirty(t *testing.T) {
	o, _ := newOverlay(t, 40)
	o.Open()

	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, o.IsOpen())
	assert.False(t, o.Dirty(), "nothing changed, so nothing to re-run")
}

// Guards against a row whose description repeats its label, which reads as noise in a list this long.
func TestOptions_DescriptionsAddInformation(t *testing.T) {
	o, _ := newOverlay(t, 40)

	for _, tg := range o.toggles {
		assert.NotEqual(t, strings.ToLower(tg.label), strings.ToLower(tg.desc),
			"row %q", tg.label)
		// The modal is as wide as its widest row, so descriptions are kept terse rather than clipped at render time.
		assert.LessOrEqual(t, len(tg.desc), 40, "row %q description is too long for the modal", tg.label)
	}
}

// TestOptions_WidthIsStableWhileScrolling pins the frame against every row rather than the visible window.
//
// The overlay windows its rows around the cursor, so without a pinned width the box resizes on every cursor move as
// the widest row in view changes - the frame twitching while the list moves under it.
func TestOptions_WidthIsStableWhileScrolling(t *testing.T) {
	o, _ := newOverlay(t, 20)
	o.Open()
	all, _ := o.lines()
	require.Greater(t, len(all), o.visibleRows(), "precondition: the list must overflow, or nothing scrolls")

	seen := map[int]int{}
	for range len(o.toggles) + 2 {
		seen[lipgloss.Width(o.View())]++
		_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	}

	assert.Len(t, seen, 1, "the frame changed width while scrolling; widths seen: %v", seen)
}
