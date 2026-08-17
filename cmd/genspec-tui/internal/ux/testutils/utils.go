// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/testloader"
)

// StripANSI removes the SGR escape sequences lipgloss emits.
//
// Assertions can then look at the text a user reads, rather than at the styling around it.
func StripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}

	return b.String()
}

func KeyRune(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

// WriteTempGo puts a Go file on disk and returns its path.
func WriteTempGo(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "x.go")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

// ApplyLoader writes the loader this run was asked for onto opts, and returns it for chaining.
//
// The TUI scans through the same library as everything else, so its tests answer to the same
// run-wide setting; see internal/testloader for what selects it and why.
//
// A test that pins a loader must not route through here - there is no telling a field left at its
// zero value from one set on purpose - which is why this refuses one rather than writing over it.
func ApplyLoader(opts *codescan.Options) *codescan.Options {
	if opts == nil || opts.FS != nil {
		return opts
	}
	if opts.CompiledDependencies || opts.ToolchainFreeLoader {
		panic("testutils: these options already pin a loader, so they must not be routed through " +
			"ApplyLoader; drop the call and leave the pin, or drop the pin and keep the call")
	}

	switch selected := testloader.Selected(); selected {
	case testloader.LoaderCompiled:
		opts.CompiledDependencies = true
	case testloader.LoaderOwn:
		opts.ToolchainFreeLoader = true
	case testloader.LoaderSource:
	default:
		panic("testutils: no rule for loader " + string(selected))
	}

	return opts
}
