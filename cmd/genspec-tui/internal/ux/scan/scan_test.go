// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Tests for what a scan may do to the configuration it was handed.

package scan

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan"
)

// A scan collects diagnostics and provenance through callbacks it installs on the options - and the options it is
// handed are the model's own, which the options overlay writes to and a second, overlapping scan reads.
func TestDoLeavesTheCallersOptionsAlone(t *testing.T) {
	t.Parallel()

	cfg := &codescan.Options{WorkDir: t.TempDir(), Packages: []string{"./..."}}

	_ = Do(cfg, nil)

	assert.Nil(t, cfg.OnDiagnostic, "the run collects what it reports through its own copy")
	assert.Nil(t, cfg.OnProvenance, "and what it located, too")
}

// The command returned by Run executes on a goroutine of its own, so the copy has to be taken before it exists: one
// taken inside would be reading the options at the same time as the event loop writes them.
func TestRunSnapshotsTheOptionsBeforeItRuns(t *testing.T) {
	t.Parallel()

	cfg := &codescan.Options{WorkDir: "/nonexistent-at-launch", Packages: []string{"./..."}}

	cmd := Run(cfg, nil)

	// As toggling a row in the options overlay does, while the scan is in flight.
	cfg.WorkDir = "/nonexistent-after-launch"

	res, ok := cmd().(ResultMsg)
	require.True(t, ok)
	require.Error(t, res.Err)

	assert.Contains(t, res.Err.Error(), "/nonexistent-at-launch",
		"a scan runs on the options it was started with, not on whatever they became meanwhile")
}
