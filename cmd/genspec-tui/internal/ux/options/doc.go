// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package options is the scanner-options overlay: a scrollable modal of boolean toggles bound directly to the
// codescan.Options the app scans with.
//
// It follows the same contract as the panels and the help overlay - a concrete type the root model owns and drives,
// never a tea.Model. It records what the user asked for and reports it as dirty; deciding that a rescan is how a
// change takes effect is the root model's business, not the modal's.
package options
