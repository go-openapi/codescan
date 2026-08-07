// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package scan runs codescan and reports the outcome as a bubbletea message.
//
// [Run] returns the command the model issues; [Do] is the same work synchronously, for tests and for callers that
// already have a goroutine. Both render the spec twice, as JSON and as YAML, so a format toggle costs no rescan.
//
// A scan never fails silently: a hard error and the soft diagnostics both ride back on [ResultMsg], since the pane has
// to show something either way.
package scan
