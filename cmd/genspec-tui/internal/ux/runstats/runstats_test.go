// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runstats_test

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/runstats"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
)

const mb = 1 << 20

// aRun is a plausible reading, with round figures so the card's own formatting is what the assertions read.
func aRun() scan.Cost {
	return scan.Cost{
		Measured:  true,
		ScanFor:   2 * time.Second,
		RenderFor: 92 * time.Millisecond,

		AllocScan:   356 * mb,
		AllocRender: 56 * mb,

		RetainScan:   31 * mb,
		RetainRender: 7 * mb,

		LiveBefore: 94 * mb,
		LiveAfter:  132 * mb,

		Objects:  410_000,
		GCCycles: 7,
		Sys:      221 * mb,
	}
}

func view(t *testing.T, o *runstats.Overlay) string {
	t.Helper()

	o.SetSize(100, 40)

	return testutils.StripANSI(o.View())
}

func TestOverlayReportsTheRun(t *testing.T) {
	o := runstats.New()
	o.Set(aRun(), false)

	got := view(t, &o)

	assert.Contains(t, got, "Last run")
	assert.Contains(t, got, "2s", "the wall clock")
	assert.Contains(t, got, "rendering 92ms")

	assert.Contains(t, got, "412.0 MB", "allocated is the two halves together")
	assert.Contains(t, got, "356.0 MB")
	assert.Contains(t, got, "56.0 MB")

	assert.Contains(t, got, "+38.0 MB", "retained carries its sign")
	assert.Contains(t, got, "94.0 MB → 132.0 MB live")

	assert.Contains(t, got, "+410 k")
	assert.Contains(t, got, "221.0 MB", "what the process holds from the OS")
}

// The figures are read here and nowhere else, so what they do not mean has to be on the card.
func TestOverlayDisclosesWhatItMeasured(t *testing.T) {
	o := runstats.New()

	o.Set(aRun(), false)
	first := view(t, &o)
	assert.Contains(t, first, "other goroutines", "the window is process-wide")
	assert.NotContains(t, first, "replaced a spec", "the first run replaced nothing")

	o.Set(aRun(), true)
	assert.Contains(t, view(t, &o), "replaced a spec",
		"a rescan was measured while the previous document was still live")
}

func TestOverlayBeforeAnyRun(t *testing.T) {
	o := runstats.New()

	got := view(t, &o)

	assert.Contains(t, got, "no run measured yet")
	assert.NotContains(t, got, "allocated", "a zero cost must not be shown as a measurement")
}

func TestOverlayOpensAndCloses(t *testing.T) {
	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyEsc},
		testutils.KeyRune('m'),
		{Type: tea.KeyEnter},
	} {
		o := runstats.New()
		o.Open()
		require.True(t, o.IsOpen())

		_ = o.HandleKey(msg)

		assert.False(t, o.IsOpen(), "%v closes the card", msg)
	}
}

// The card covers the UI, so a key it does not own is swallowed rather than acted on behind it.
func TestOverlaySwallowsOtherKeys(t *testing.T) {
	o := runstats.New()
	o.Open()

	for _, msg := range []tea.KeyMsg{testutils.KeyRune('r'), testutils.KeyRune('j'), {Type: tea.KeyF3}} {
		assert.Nil(t, o.HandleKey(msg))
		assert.True(t, o.IsOpen(), "%v left the card open", msg)
	}
}

// The frame is pinned to the widest line it can show, so the box does not resize as the figures change.
func TestOverlayFrameIsStable(t *testing.T) {
	o := runstats.New()
	o.Set(aRun(), false)
	narrow := width(view(t, &o))

	big := aRun()
	big.AllocScan = 3 * 1024 * mb
	o.Set(big, true)

	assert.GreaterOrEqual(t, width(view(t, &o)), narrow, "a wider figure widens the frame rather than clipping")
}

func width(s string) int {
	w := 0
	for line := range strings.SplitSeq(s, "\n") {
		w = max(w, len([]rune(line)))
	}

	return w
}
