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
	o.Set(aRun(), nil, false)

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

// The line a reader glances at: which phase the run went on, in both currencies at once.
func TestOverlayRecapsTheSplit(t *testing.T) {
	o := runstats.New()
	o.Set(aRun(), nil, false)

	got := view(t, &o)

	assert.Contains(t, got, "time 96% / 4%", "2s scanning against 92ms rendering")
	assert.Contains(t, got, "memory 86% / 14%", "356 MB against 56 MB")
	assert.Contains(t, got, "scanning / rendering", "which way round the pairs read")
	assert.NotContains(t, got, "cpu ", "there is no CPU account without a profile")
}

// The sampled CPU split joins the recap only when there is one, and is the honest clock: wall time counts waiting.
func TestOverlayRecapsTheSampledCPU(t *testing.T) {
	o := runstats.New()
	o.Set(aRun(), aProfile(), false)
	o.SetSize(160, 120)

	assert.Contains(t, testutils.StripANSI(o.View()), "cpu 96% / 4%", "2s of samples against 90ms")
}

// Halves rounded independently produce "96% / 5%", which reads as an arithmetic error in a line meant to be glanced at.
func TestOverlayRecapAddsUp(t *testing.T) {
	run := aRun()
	run.ScanFor, run.RenderFor = 955*time.Millisecond, 45*time.Millisecond
	run.AllocScan, run.AllocRender = 0, 0

	o := runstats.New()
	o.Set(run, nil, false)

	got := view(t, &o)

	assert.Contains(t, got, "time 96% / 4%")
	assert.Contains(t, got, "memory n/a", "a phase pair that allocated nothing has no ratio to report")
}

// The figures are read here and nowhere else, so what they do not mean has to be on the card.
func TestOverlayDisclosesWhatItMeasured(t *testing.T) {
	o := runstats.New()

	o.Set(aRun(), nil, false)
	first := view(t, &o)
	assert.Contains(t, first, "other goroutines", "the window is process-wide")
	assert.NotContains(t, first, "replaced a spec", "the first run replaced nothing")

	o.Set(aRun(), nil, true)
	assert.Contains(t, view(t, &o), "replaced a spec",
		"a rescan was measured while the previous document was still live")
}

func TestOverlayBeforeAnyRun(t *testing.T) {
	o := runstats.New()

	got := view(t, &o)

	assert.Contains(t, got, "no run measured yet")
	assert.NotContains(t, got, "allocated", "a zero cost must not be shown as a measurement")
}

// aProfile is what a profiled run hands over: who spent the CPU, who asked for the memory, and where the artifacts are.
func aProfile() *scan.ProfileReport {
	return &scan.ProfileReport{
		Dir:      "/tmp/genspec-tui-profile-42",
		MemPaths: []string{"/tmp/p/mem-before.pprof", "/tmp/p/mem-after-scan.pprof", "/tmp/p/mem-after-render.pprof"},
		Scan: scan.PhaseProfile{
			Label:      "scanning",
			CPUPath:    "/tmp/genspec-tui-profile-42/cpu-scan.pprof",
			CPUTotal:   2 * time.Second,
			CPUSamples: 200,
			CPU: []scan.Func{
				{Name: "go/types.(*Checker).checkFiles", Flat: 900 * time.Millisecond, Share: 0.45},
				{
					Name:  "github.com/go-openapi/codescan/internal/builders/schema.(*Builder).Build",
					Flat:  time.Second / 2,
					Share: 0.25,
				},
			},
			Alloc: []scan.Site{{Name: "go/parser.ParseFile", Bytes: 182 * mb, Objects: 410_000}},
		},
		Render: scan.PhaseProfile{
			Label:      "rendering",
			CPUPath:    "/tmp/genspec-tui-profile-42/cpu-render.pprof",
			CPUTotal:   90 * time.Millisecond,
			CPUSamples: 9,
			CPU:        []scan.Func{{Name: "encoding/json.Marshal", Flat: 90 * time.Millisecond, Share: 1}},
			Alloc: []scan.Site{
				{Name: "encoding/json.Marshal", Bytes: 48 * mb, Objects: 12_000},
				{Name: "go.yaml.in/yaml/v3.yaml_emitter_emit", Bytes: 5 * mb, Objects: 21},
			},
		},
	}
}

func TestOverlayReportsAProfiledRun(t *testing.T) {
	o := runstats.New()
	o.Set(aRun(), aProfile(), false)
	o.SetSize(160, 120) // tall enough that nothing is below the fold

	got := testutils.StripANSI(o.View())

	assert.Contains(t, got, "Last run · profiled")
	assert.Contains(t, got, "profiler's overhead", "the summary figures now carry the observer's cost")
	assert.NotContains(t, got, "other goroutines",
		"the process-wide caveat is answered by the tables: the noise arrives named")

	assert.Contains(t, got, "where the CPU went — scanning")
	assert.Contains(t, got, "where the CPU went — rendering", "both phases are profiled, and both are reported")
	assert.Contains(t, got, "types.(*Checker).checkFiles", "the repository path is trimmed off the symbol")
	assert.NotContains(t, got, "github.com/go-openapi/codescan/internal/builders")
	assert.Contains(t, got, "schema.(*Builder).Build")
	assert.Contains(t, got, "45%")

	assert.Contains(t, got, "what allocated it — scanning")
	assert.Contains(t, got, "182.0 MB")
	assert.Contains(t, got, "what allocated it — rendering")
	assert.Contains(t, got, "yaml/v3.yaml_emitter_emit",
		"a major-version segment keeps the package before it: v3.yaml_emitter_emit names no library")
	assert.Contains(t, got, "estimated from sampling")

	assert.Contains(t, got, "too few to rank functions",
		"nine samples of rendering is a hint, and says so; two hundred of scanning does not")
	assert.Contains(t, got, "/tmp/genspec-tui-profile-42/cpu-scan.pprof")
	assert.Contains(t, got, "/tmp/genspec-tui-profile-42/cpu-render.pprof")
	assert.Contains(t, got, "go tool pprof -http=: -base /tmp/p/mem-before.pprof /tmp/p/mem-after-scan.pprof",
		"the command that reproduces the scanning phase in the real tool")
}

// Exactness is a property of the run's sampling rate, and the reader has to be told which one they are looking at.
func TestOverlaySaysWhetherAllocationsWereCounted(t *testing.T) {
	o := runstats.New()
	report := aProfile()
	report.Exact = true
	o.Set(aRun(), report, false)
	o.SetSize(160, 120)

	assert.Contains(t, testutils.StripANSI(o.View()), "every allocation counted")
}

// A phase the sampler never caught says so, rather than showing an empty table that reads as "nothing allocated".
func TestOverlayOnAnEmptyPhase(t *testing.T) {
	o := runstats.New()
	report := aProfile()
	report.Render.Alloc = nil
	o.Set(aRun(), report, false)
	o.SetSize(160, 120)

	assert.Contains(t, testutils.StripANSI(o.View()), "nothing sampled in this phase")
}

// The tables do not fit a terminal, and a table below the fold is a measurement nobody knows was taken.
func TestOverlayScrolls(t *testing.T) {
	o := runstats.New()
	o.Set(aRun(), aProfile(), false)
	o.SetSize(120, 24)

	head := testutils.StripANSI(o.View())
	assert.Contains(t, head, " of ", "the footer says how much is below")
	assert.NotContains(t, head, "cpu-scan.pprof", "the artifacts are further down")

	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyEnd})
	tail := testutils.StripANSI(o.View())

	assert.NotEqual(t, head, tail, "End moved the window")
	assert.Contains(t, tail, "cpu-scan.pprof")
	assert.Equal(t, width(head), width(tail), "the frame is pinned, so the box does not twitch as it scrolls")
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
	o.Set(aRun(), nil, false)
	narrow := width(view(t, &o))

	big := aRun()
	big.AllocScan = 3 * 1024 * mb
	o.Set(big, nil, true)

	assert.GreaterOrEqual(t, width(view(t, &o)), narrow, "a wider figure widens the frame rather than clipping")
}

func width(s string) int {
	w := 0
	for line := range strings.SplitSeq(s, "\n") {
		w = max(w, len([]rune(line)))
	}

	return w
}
