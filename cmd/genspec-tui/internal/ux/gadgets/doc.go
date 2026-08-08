// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package gadgets holds small, self-contained TUI helpers.
//
// The clipboard helper is ported from fredbi/git-janitor: it copies text reliably across terminals by trying real
// clipboard tools first (which report success), then falling back to OSC 52 escape sequences (which work over SSH and
// in modern terminals without any external tool), with tmux passthrough wrapping.
package gadgets
