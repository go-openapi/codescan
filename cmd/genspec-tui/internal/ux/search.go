// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SearchBox is the spec-pane search prompt: the text input, and whether it currently holds the keyboard.
//
// While it is active it takes the status line and every key, which is why "is the user typing a query" has to be
// answerable from outside - but nothing outside needs to know how the input works.
type SearchBox struct {
	active bool
	input  textinput.Model
}

// NewSearchBox builds a closed search prompt.
func NewSearchBox() SearchBox {
	in := textinput.New()
	in.Prompt = "/"
	in.Placeholder = "search spec"

	return SearchBox{input: in}
}

// Active reports whether the prompt is capturing input.
func (s *SearchBox) Active() bool { return s.active }

// Open clears the prompt and gives it the keyboard.
//
// Always from empty: a search is started to look for something, not to resume the last one.
func (s *SearchBox) Open() tea.Cmd {
	s.active = true
	s.input.SetValue("")

	return s.input.Focus()
}

// Close dismisses the prompt, releasing the keyboard.
func (s *SearchBox) Close() {
	s.active = false
	s.input.Blur()
}

// Query holds the text the user has typed.
func (s *SearchBox) Query() string { return s.input.Value() }

// Update forwards a key to the input.
func (s *SearchBox) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)

	return cmd
}

// View renders the prompt, which occupies the status line while it is active.
func (s *SearchBox) View() string { return s.input.View() }
