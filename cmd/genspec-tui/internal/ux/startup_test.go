// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Tests for what the command settles before the UI exists, and what the session says about it.

package ux

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
)

// A session that behaves unlike the command line says it should is a session whose configuration file nobody knows
// about. It is said once, on the status line, and expires like every other notice.
func TestStartup_AnnouncesTheConfigurationFile(t *testing.T) {
	m := New(Startup{
		Options:    codescan.Options{WorkDir: t.TempDir(), Packages: []string{"./..."}},
		ConfigPath: "/proj/.codescan.yaml",
		ConfigSet:  []string{"workdir", "scan-models"},
	})
	t.Cleanup(m.Close)

	require.NotNil(t, m.announceConfig())

	assert.Contains(t, m.notice, "/proj/.codescan.yaml")
	assert.Contains(t, m.notice, "2 settings")
}

func TestStartup_SaysWhenTheFileDecidedNothing(t *testing.T) {
	m := New(Startup{
		Options:    codescan.Options{WorkDir: t.TempDir()},
		ConfigPath: "/proj/.codescan.yaml",
	})
	t.Cleanup(m.Close)

	require.NotNil(t, m.announceConfig())

	assert.Contains(t, m.notice, "it set nothing",
		"a file that was read and changed nothing is worth distinguishing from no file at all")
}

// One setting is one setting.
func TestStartup_CountsOneSettingSingly(t *testing.T) {
	m := New(Startup{
		Options:    codescan.Options{WorkDir: t.TempDir()},
		ConfigPath: "/proj/.codescan.yaml",
		ConfigSet:  []string{"workdir"},
	})
	t.Cleanup(m.Close)

	_ = m.announceConfig()

	assert.Contains(t, m.notice, "1 setting")
	assert.NotContains(t, m.notice, "1 settings")
}

// The ordinary case: no file, nothing said.
func TestStartup_SaysNothingWithoutAFile(t *testing.T) {
	m := New(Startup{Options: codescan.Options{WorkDir: t.TempDir()}})
	t.Cleanup(m.Close)

	assert.Nil(t, m.announceConfig())
	assert.Empty(t, m.notice)
}

// Profiling travels on the same struct, and reaches the scans the session runs.
func TestStartup_CarriesProfilingToTheScans(t *testing.T) {
	dir := t.TempDir()
	m := New(Startup{
		Options:   codescan.Options{WorkDir: dir},
		Profiling: scan.Profiling{Enabled: true, Dir: dir},
	})
	t.Cleanup(m.Close)

	assert.True(t, m.profiling.Enabled)
	assert.Equal(t, dir, m.profiling.Dir)
}
