// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain forces a colour profile for the whole package's tests.
//
// The reason is that lipgloss degrades to plain text off a TTY, which go test never is,
// so without this every style renders identically and any assertion about styling passes just as happily against
// a pane that applied none.
//
// The panels package needs the same thing for the same reason.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}
