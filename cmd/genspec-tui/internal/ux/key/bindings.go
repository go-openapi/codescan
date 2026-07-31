// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package key normalizes tea.KeyMsg values into a small enum of named
// bindings, so the model dispatches on a plain string switch rather than a
// key-binding library. Mirrors the convention in fredbi/git-janitor.
package key

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Binding is a lowercased key descriptor (e.g. "ctrl+j", "tab", "k").
type Binding string

const (
	CtrlC    Binding = "ctrl+c"
	CtrlQ    Binding = "ctrl+q"
	CtrlJ    Binding = "ctrl+j"
	CtrlY    Binding = "ctrl+y"
	Tab      Binding = "tab"
	ShiftTab Binding = "shift+tab"
	Up       Binding = "up"
	Down     Binding = "down"
	Left     Binding = "left"
	Right    Binding = "right"
	J        Binding = "j"
	K        Binding = "k"
	H        Binding = "h"
	L        Binding = "l"
	C        Binding = "c"
	R        Binding = "r"
	O        Binding = "o"
	G        Binding = "g"
	F        Binding = "f"
	I        Binding = "i"
	PgUp     Binding = "pgup"
	PgDown   Binding = "pgdown"
	Home     Binding = "home"
	End      Binding = "end"
	Space    Binding = " "
	Question Binding = "?"
	Esc      Binding = "esc"
	Enter    Binding = "enter"

	// F3 steps to the next reference of the definition under the spec cursor.
	F3 Binding = "f3"

	// ShiftF3 steps to the PREVIOUS reference.
	//
	// bubbletea v1's Key carries no Shift modifier, and the xterm family maps
	// shift+F1..F12 onto F13..F24 — so shift+F3 reaches us as F15. This is
	// terminal-dependent: a terminal that emits nothing distinguishable for
	// shift+F3 simply has no prev key.
	ShiftF3 Binding = "f15"

	// ShiftF3Named is the literal spelling, accepted so a terminal (or a future
	// bubbletea) that reports the modifier directly also works.
	ShiftF3Named Binding = "shift+f3"
)

// MsgBinding normalizes a key message to a Binding.
func MsgBinding(msg tea.KeyMsg) Binding {
	return Binding(strings.ToLower(msg.String()))
}

// Quit reports whether the binding requests application exit.
func (b Binding) Quit() bool { return b == CtrlC || b == CtrlQ }
