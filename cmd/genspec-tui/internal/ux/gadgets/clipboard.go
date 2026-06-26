// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package gadgets holds small, self-contained TUI helpers. The clipboard
// helper is ported from fredbi/git-janitor: it copies text reliably across
// terminals by trying real clipboard tools first (which report success), then
// falling back to OSC 52 escape sequences (which work over SSH and in modern
// terminals without any external tool), with tmux passthrough wrapping.
package gadgets

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CopyToClipboard copies text to the system clipboard.
//
// It tries command-line tools first — they give reliable feedback (OSC 52 is
// fire-and-forget, so we can't tell whether the terminal honored it) — then
// falls back to OSC 52.
func CopyToClipboard(ctx context.Context, text string) error {
	if err := clipboardViaTool(ctx, text); err == nil {
		return nil
	}

	return osc52Copy(text)
}

// clipboardViaTool tries xclip, xsel, then wl-copy, in order.
func clipboardViaTool(ctx context.Context, text string) error {
	tools := []struct {
		name string
		args []string
	}{
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
		{"wl-copy", nil},
	}

	for _, t := range tools {
		path, err := exec.LookPath(t.name)
		if err != nil {
			continue
		}

		cmd := exec.CommandContext(ctx, path, t.args...)
		cmd.Stdin = strings.NewReader(text)

		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return errors.New("no clipboard tool available (tried xclip, xsel, wl-copy)")
}

// osc52Copy writes an OSC 52 escape sequence to stderr, instructing the
// terminal emulator to copy text to the system clipboard. Works on kitty,
// alacritty, wezterm, iTerm2, Windows Terminal, foot, etc.; not on
// gnome-terminal or some older terminals. Stderr is used so the sequence
// bypasses bubbletea's stdout render buffer.
func osc52Copy(text string) error {
	b64 := base64.StdEncoding.EncodeToString([]byte(text))

	// OSC 52 ; c ; <base64> BEL
	seq := fmt.Sprintf("\x1b]52;c;%s\x07", b64)

	// Detect tmux and wrap in passthrough DCS.
	if isTmux() {
		seq = fmt.Sprintf("\x1bPtmux;\x1b%s\x1b\\", seq)
	}

	_, err := fmt.Fprint(os.Stderr, seq)

	return err
}

func isTmux() bool {
	return strings.HasPrefix(os.Getenv("TERM_PROGRAM"), "tmux") ||
		os.Getenv("TMUX") != ""
}
