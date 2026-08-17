// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package safetext makes the control characters in scanned text visible, so nothing a repository contains can steer
// the terminal the TUI is drawn on.
//
// Everything the panes show is attacker-controlled to some degree: a file's contents, the names of the files and
// directories around it, and the diagnostics and validation findings quoting all three. Written to a terminal
// verbatim, an ESC in any of those is not text - it is a command, and it repaints, retitles or overwrites whatever the
// operator was reading.
//
// So no external string reaches a styling call unescaped. [Sanitize] is the default and encodes every control
// character; [SanitizeBlock] is the exception for prose a pane lays out itself, and spares only the line feed and the
// tab.
//
// Both replace one rune with exactly one rune, which is what lets them sit under the renderers without disturbing
// them: the source viewer addresses syntax runs, diagnostic marks and mouse clicks by rune column, and a substitution
// that changed a line's length would move every one of them.
package safetext
