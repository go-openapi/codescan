// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package watcher reports changes under a source tree, so the spec can be regenerated on save.
//
// [New] walks the tree once and watches every directory in it, skipping the ones a scan would never read. It is
// best effort by design: when a watcher cannot be created the caller falls back to rescanning on demand, so the TUI
// stays usable on platforms and filesystems where watching does not work.
package watcher
