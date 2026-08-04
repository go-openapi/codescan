// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package ux is the bubbletea front-end for genspec-tui: a single root Model composing a header line, three panels
// (source tree, spec, diagnostics), and a status/help line.
//
// Structure borrows from fredbi/git-janitor — one root model owning panel values, an enum-based key dispatch, mouse
// focus/scroll, and a recalcLayout that distributes the terminal size across panels.
//
// # What lives where
//
// The panels ([panels]) and the modal overlays ([help], [options]) are separate packages because each owns its own
// state and needs almost nothing from the model. What is left here does not split that way: the cross-reference layer
// alone reaches into all three panels, both spec indexes, the source index and the status line, so it is spread across
// files rather than hidden behind a package boundary it could not honestly keep.
//
//   - model.go     the Model struct, its lifecycle (New/Close/Init/Update/View), layout and focus routing
//   - keys.go      key dispatch: global bindings, per-pane handlers, the search input, the editor
//   - crossref.go  follow modes, go-to-definition, find-references, gutters, the spec render
//   - refcycle.go  the find-references walk, as its own small type
//   - source.go    the open file: loading, saving, syntax runs, diagnostic marks, buffer coordinates
//   - scanflow.go  running a scan and absorbing its result, the file watcher's debounce, transient notices
//   - chrome.go    the header, status line and follow badge
//   - mouse.go     click-to-focus and wheel scrolling
//   - overlay.go   the Overlay contract the modals satisfy, and their precedence
package ux
