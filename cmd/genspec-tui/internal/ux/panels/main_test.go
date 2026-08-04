// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain forces a colour profile for the whole package's tests.
//
// lipgloss degrades to plain text when stdout is not a TTY, which `go test` never is — so without this every style
// renders identically and the panels' visual contracts (driver bar vs follower tint, §6.5) would be unfalsifiable: the
// assertions would pass just as happily against a panel that applied no style at all.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}
