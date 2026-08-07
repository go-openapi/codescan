// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package diagnostics renders the scan half of the diagnostics pane: a severity tally, then one row per finding.
//
// It is presentation only. The model owns the diagnostics and decides which one is selected; this package turns that
// into text, and reports back which content line the selection landed on so the pane can scroll to it.
//
// Rows are coloured by severity, and the selected row takes the whole line instead. Those are alternatives rather than
// layers: a highlight laid over already-coloured text is closed early by the inner style's reset, which breaks the bar
// partway across.
package diagnostics
