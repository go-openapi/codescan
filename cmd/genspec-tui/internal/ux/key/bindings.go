// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

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
	M        Binding = "m"
	N        Binding = "n"
	V        Binding = "v"
	Y        Binding = "y"
	PgUp     Binding = "pgup"
	PgDown   Binding = "pgdown"
	Home     Binding = "home"
	End      Binding = "end"
	Space    Binding = " "
	Question Binding = "?"
	Esc      Binding = "esc"
	Enter    Binding = "enter"

	// CtrlUp / CtrlDown / CtrlLeft / CtrlRight move the pane dividers, each in the arrow's own direction.
	//
	// Terminal-dependent, like ShiftF3: the xterm family reports these as CSI 1;5<A-D> and most modern emulators follow,
	// but a terminal that sends a bare arrow instead simply has no resize keys - nothing else misfires, because a bare
	// arrow is already a navigation key.
	CtrlUp    Binding = "ctrl+up"
	CtrlDown  Binding = "ctrl+down"
	CtrlLeft  Binding = "ctrl+left"
	CtrlRight Binding = "ctrl+right"

	// F3 steps to the next reference of the definition under the spec cursor.
	F3 Binding = "f3"

	// F5 re-reads the open file from disk, the browser-standard spelling of "reload".
	//
	// Deliberately not also ctrl+r: the editor is a live textarea, and ctrl+r is redo or reverse-search in enough editors
	// that binding it to a discarding action would be a trap.
	F5 Binding = "f5"

	// ShiftF3 steps to the PREVIOUS reference.
	//
	// bubbletea v1's Key carries no Shift modifier, and the xterm family maps shift+F1..F12 onto F13..F24 - so shift+F3
	// reaches us as F15. This is terminal-dependent: a terminal that emits nothing distinguishable for shift+F3 simply has
	// no prev key.
	ShiftF3 Binding = "f15"

	// ShiftF3Named is the literal spelling, accepted so a terminal (or a future bubbletea) that reports the modifier
	// directly also works.
	ShiftF3Named Binding = "shift+f3"
)

// MsgBinding normalizes a key message to a Binding.
func MsgBinding(msg tea.KeyMsg) Binding {
	return Binding(strings.ToLower(msg.String()))
}

// Quit reports whether the binding requests application exit.
func (b Binding) Quit() bool { return b == CtrlC || b == CtrlQ }

// Nav maps a movement binding to a signed delta.
//
// page is how far a page key travels; span is the full extent of whatever is being moved over - which is what lets
// Home and End be plain deltas too. Every caller already clamps, so ∓span lands exactly on the ends, and one rule
// then covers a scroll offset, a list index and a line cursor alike. Six panes had each spelled this out on their
// own, and had drifted into three different ways of saying "go to the top".
//
// Reports ok=false for anything that is not a movement, so a caller can fall through to the keys it owns.
func Nav(b Binding, page, span int) (delta int, ok bool) {
	switch b {
	case Up, K:
		return -1, true
	case Down, J:
		return +1, true
	case PgUp:
		return -page, true
	case PgDown:
		return +page, true
	case Home:
		return -span, true
	case End:
		return +span, true
	default:
		return 0, false
	}
}
