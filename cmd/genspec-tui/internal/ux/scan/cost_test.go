// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestCostOf(t *testing.T) {
	before := memFence{total: 1_000, live: 500, objects: 40, gc: 3, sys: 9_000}
	scanned := memFence{total: 5_000, live: 2_500, objects: 190, gc: 6, sys: 12_000}
	rendered := memFence{total: 6_000, live: 3_000, objects: 210, gc: 7, sys: 13_000}

	c := costOf(before, scanned, rendered, 2*time.Second, 100*time.Millisecond)

	require.True(t, c.Measured)
	assert.Equal(t, 2*time.Second, c.ScanFor)
	assert.Equal(t, 100*time.Millisecond, c.RenderFor)

	assert.Equal(t, uint64(4_000), c.AllocScan)
	assert.Equal(t, uint64(1_000), c.AllocRender)
	assert.Equal(t, uint64(5_000), c.Allocated(), "the halves account for the whole run's churn")

	assert.Equal(t, int64(2_000), c.RetainScan)
	assert.Equal(t, int64(500), c.RetainRender)
	assert.Equal(t, int64(2_500), c.Retained())
	assert.Equal(t, c.Retained(), diff(c.LiveAfter, c.LiveBefore),
		"the split must agree with the outer fences it is derived from")

	assert.Equal(t, int64(170), c.Objects)
	assert.Equal(t, uint32(4), c.GCCycles)
	assert.Equal(t, uint64(13_000), c.Sys, "Sys is read at the end, not differenced")
}

// A half that collects more than it allocates finishes with LESS live than it started with. Reading that as an unsigned
// difference would report an enormous positive number - the whole point of holding the retained figures signed.
func TestCostOfAcrossACollection(t *testing.T) {
	before := memFence{total: 1_000, live: 900, objects: 90}
	scanned := memFence{total: 5_000, live: 300, objects: 20}
	rendered := memFence{total: 5_500, live: 400, objects: 25}

	c := costOf(before, scanned, rendered, time.Second, time.Second)

	assert.Equal(t, int64(-600), c.RetainScan)
	assert.Equal(t, int64(100), c.RetainRender)
	assert.Equal(t, int64(-500), c.Retained())
	assert.Equal(t, int64(-65), c.Objects)
	assert.Equal(t, uint64(4_500), c.Allocated(), "churn is unaffected: TotalAlloc only ever grows")
}

// A run that failed is fenced twice at the same point, so the half that never happened reads as zero rather than as
// something the reader has to discount.
func TestCostOfAFailedRun(t *testing.T) {
	before := memFence{total: 1_000, live: 500}
	scanned := memFence{total: 3_000, live: 1_500}

	c := costOf(before, scanned, scanned, time.Second, 0)

	assert.Equal(t, uint64(2_000), c.AllocScan)
	assert.Zero(t, c.AllocRender)
	assert.Zero(t, c.RetainRender)
	assert.Equal(t, uint64(1_500), c.LiveAfter)
}

func TestZeroCostIsNotAMeasurement(t *testing.T) {
	assert.False(t, Cost{}.Measured,
		"the state before the first scan must be distinguishable from a run that cost nothing")
}

func TestDiff(t *testing.T) {
	assert.Equal(t, int64(5), diff(15, 10))
	assert.Equal(t, int64(-5), diff(10, 15))
	assert.Zero(t, diff(10, 10))
}
