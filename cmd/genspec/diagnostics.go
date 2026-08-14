// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/SladkyCitron/slogcolor"

	"github.com/go-openapi/codescan"
)

// reporter is where everything the scan observed goes.
//
// codescan never writes to standard error itself - every observation arrives through a callback - so
// this is the only thing standing between a scan and a silent one. It is also what decides the exit
// status: a caller running this in a pipeline wants to be told, and a summary a human reads is not
// something a pipeline can act on.
type reporter struct {
	logger *slog.Logger

	// root is what positions are reported relative to, so that a diagnostic names the file the way
	// the caller would.
	root string

	// hints says whether hints are reported. They are muted by default: they are the scanner
	// thinking aloud - a model it discovered, a name it reduced - and there are a great many.
	hints bool

	// threshold is the least serious severity that still fails the command, when there is one.
	threshold codescan.Severity
	failing   bool

	// seen counts every diagnostic by severity, muted ones included: what was suppressed was still
	// observed, and the summary saying so is how a caller learns -verbose has something to show.
	seen map[codescan.Severity]int

	// tripped records that something reached the threshold, so that the whole scan is reported
	// before the command exits on it rather than at the first one.
	tripped bool
}

// newReporter wires a sink for the scan's diagnostics.
func newReporter(cfg *config, stderr io.Writer) (*reporter, error) {
	threshold, failing, err := resolveFailOn(*cfg.failOn)
	if err != nil {
		return nil, err
	}

	colorize, err := resolveColor(*cfg.color, stderr)
	if err != nil {
		return nil, err
	}

	return &reporter{
		logger:    logger(stderr, colorize, *cfg.quiet),
		hints:     *cfg.verbose,
		threshold: threshold,
		failing:   failing,
		seen:      make(map[codescan.Severity]int),
	}, nil
}

// onDiagnostic is what [codescan.Options.OnDiagnostic] is given.
func (r *reporter) onDiagnostic(diag codescan.Diagnostic) {
	r.record(diag.Severity)

	if diag.Severity == codescan.SeverityHint && !r.hints {
		return
	}

	r.log(diag.Severity, diag.Message,
		slog.String("code", string(diag.Code)),
		slog.String("file", r.relative(diag.Pos.Filename)),
		slog.Int("line", diag.Pos.Line),
		slog.Int("column", diag.Pos.Column),
	)
}

// record tallies one finding and decides whether it fails the command.
//
// Everything the command reports goes through here - what the scan observed and what -validate
// found - because a reader meets them as one stream, in one set of colours, and a threshold that saw
// only half of it would be a trap rather than a policy.
//
// A severity is more serious the lower it sorts, so "at least as serious as the threshold" reads
// backwards. Counted even when the finding itself is muted: what makes a command fail should not
// depend on how loud it was asked to be.
func (r *reporter) record(severity codescan.Severity) {
	r.seen[severity]++

	if r.failing && severity <= r.threshold {
		r.tripped = true
	}
}

// log writes one line at the level a severity maps onto.
func (r *reporter) log(sev codescan.Severity, msg string, attrs ...any) {
	switch sev {
	case codescan.SeverityError:
		r.logger.Error(msg, attrs...)
	case codescan.SeverityWarning:
		r.logger.Warn(msg, attrs...)
	case codescan.SeverityHint:
		r.logger.Info(msg, attrs...)
	default:
		r.logger.Info(msg, attrs...)
	}
}

// relative renders a position's file the way the caller named it.
//
// An absolute path is what the scanner records, and what it means is right - but it is not what
// somebody standing in their own module wants to read, nor what an editor can be handed. Where the
// two cannot be related, the absolute path is still better than nothing.
func (r *reporter) relative(filename string) string {
	if filename == "" || r.root == "" {
		return filename
	}

	rel, err := filepath.Rel(r.root, filename)
	if err != nil {
		return filename
	}

	return rel
}

// summarize reports what the command observed in all, and whether that is a failure.
//
// The counts include what was muted. A scan whose hints were suppressed says how many there were,
// which is the only way a caller finds out that -verbose has anything to add.
func (r *reporter) summarize() error {
	counted := []struct {
		severity codescan.Severity
		label    string
	}{
		{codescan.SeverityError, "errors"},
		{codescan.SeverityWarning, "warnings"},
		{codescan.SeverityHint, "hints"},
	}

	attrs := make([]any, 0, len(counted))
	total := 0
	for _, c := range counted {
		if n := r.seen[c.severity]; n > 0 {
			attrs = append(attrs, slog.Int(c.label, n))
			total += n
		}
	}

	if total > 0 {
		r.logger.Info("finished", attrs...)
	}

	if r.tripped {
		return fmt.Errorf("%w (%s)", errDiagnostics, r.threshold)
	}

	return nil
}

// logger builds the sink the reporter writes through.
func logger(stderr io.Writer, colorize, quiet bool) *slog.Logger {
	if quiet {
		return slog.New(discardHandler{})
	}

	opts := *slogcolor.DefaultOptions
	opts.NoColor = !colorize
	// A timestamp on every line of a scan that takes a second, and the name of the file inside this
	// command that happened to call the logger, are both noise: what a reader wants located is the
	// diagnostic's own position, which travels as an attribute.
	opts.NoTime = true
	opts.SrcFileMode = slogcolor.Nop

	return slog.New(slogcolor.NewHandler(stderr, &opts))
}

// resolveColor decides whether to colorize, from what was asked and where it is writing.
func resolveColor(mode string, stderr io.Writer) (bool, error) {
	switch mode {
	case colorAlways:
		return true, nil
	case colorNever:
		return false, nil
	case colorAuto:
		return isTerminal(stderr) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb", nil
	default:
		return false, fmt.Errorf("%w: -color %q is not one of %s, %s, %s",
			errUsage, mode, colorAlways, colorNever, colorAuto)
	}
}

// isTerminal reports whether w is something a person is looking at.
//
// A character device is the stdlib's own answer to that question, and it is enough here: the cost of
// being wrong is escape codes in a file, which is exactly what -color=never is for.
func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
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
