// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain forces a colour profile for the whole package's tests, matching ux and ux/panels.
//
// lipgloss degrades to plain text when stdout is not a TTY, which go test never is - so without this every style
// renders identically and the severity colouring is unfalsifiable. It is also what let a selected row's highlight
// break at the severity label unnoticed: with no escapes emitted at all, the nesting that causes it left no trace for a
// test to catch.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}
