// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import (
	"context"
	"io"
	"log/slog"

	"github.com/SladkyCitron/slogcolor"
)

// logger builds the sink the reporter writes through.
func logger(stderr io.Writer, colorize, quiet bool) *slog.Logger {
	if quiet || stderr == nil {
		return slog.New(discardHandler{})
	}

	opts := *slogcolor.DefaultOptions
	opts.NoColor = !colorize
	// discard timestamp, which is considered noise in a CLI context
	opts.NoTime = true
	opts.SrcFileMode = slogcolor.Nop

	return slog.New(slogcolor.NewHandler(stderr, &opts))
}

// discardHandler is what -quiet installs.
//
// A handler that refuses every level, rather than a writer pointed at io.Discard: the difference is
// that nothing is formatted at all, and formatting is most of the cost of a diagnostic nobody reads.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (h discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return h }
func (h discardHandler) WithGroup(string) slog.Handler           { return h }
