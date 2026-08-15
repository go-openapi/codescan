// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec/internal/sentinel"
	"github.com/go-openapi/codescan/cmd/internal/cliconf"
)

// Reporter is where everything the scan observed goes.
//
// codescan never writes to standard error itself - every observation arrives through a callback - so
// this is the only thing standing between a scan and a silent one. It is also what decides the exit
// status: a caller running this in a pipeline wants to be told, and a summary a human reads is not
// something a pipeline can act on.
type Reporter struct {
	// Root is what positions are reported relative to, so that a diagnostic names the file the way
	// the caller would.
	Root string

	logger *slog.Logger

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

// ReporterConfig holds what part of the CLI config is of interest to a [Reporter].
type ReporterConfig struct {
	Quiet         bool
	Verbose       bool
	Color         bool
	FailThreshold codescan.Severity
	Failing       bool
	Stderr        io.Writer
	Output        string
}

// NewReporter wires a sink for the scan's diagnostics.
func NewReporter(cfg ReporterConfig) *Reporter {
	return &Reporter{
		logger:    logger(cfg.Stderr, cfg.Color, cfg.Quiet),
		hints:     cfg.Verbose,
		threshold: cfg.FailThreshold,
		failing:   cfg.Failing,
		seen:      make(map[codescan.Severity]int),
	}
}

// AboutConfiguration reports what a configuration file decided, under -verbose.
//
// It reports whether a config file was read at all, the keys it skipped and if a section was misspelled.
func (r *Reporter) AboutConfiguration(path string, applied cliconf.Result) {
	if path == "" || !r.hints {
		return
	}

	r.logger.Info("configuration read",
		slog.String("file", r.relative(path)),
		slog.Int("flags", len(applied.Set)),
	)

	if len(applied.Ignored) > 0 {
		r.logger.Info("configuration keys skipped, in sections this command does not know",
			slog.String("keys", strings.Join(applied.Ignored, ", ")),
		)
	}
}

// Summarize reports what the command observed in all, and whether that is a failure.
//
// The counts include what was muted.
// A scan whose hints were suppressed says how many there were, so a caller knows if going verbose has anything to add.
func (r *Reporter) Summarize() error {
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
		return fmt.Errorf("%w (%s)", sentinel.ErrDiagnostics, r.threshold)
	}

	return nil
}

// OnDiagnostic is what [codescan.Options.OnDiagnostic] is given.
func (r *Reporter) OnDiagnostic(diag codescan.Diagnostic) {
	r.record(diag.Severity)

	if diag.Severity == codescan.SeverityHint && !r.hints {
		return
	}

	r.log(diag.Severity, diag.Message, r.attrs(diag)...)
}

// attrs is what is worth saying about a diagnostic besides its message.
//
// A diagnostic with no position - a whole route omitted by a tag rule, a definition pruned - is
// reported without one, rather than with the zero value dressed up as a location. "file= line=0" is
// not somewhere a reader can go.
func (r *Reporter) attrs(diag codescan.Diagnostic) []any {
	attrs := make([]any, 0, 4) //nolint:mnd // a code, and the three parts of a position
	attrs = append(attrs, slog.String("code", string(diag.Code)))
	if diag.Pos.Filename == "" {
		return attrs
	}

	return append(attrs,
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
func (r *Reporter) record(severity codescan.Severity) {
	r.seen[severity]++

	if r.failing && severity <= r.threshold {
		r.tripped = true
	}
}

// log writes one line at the level a severity maps onto.
func (r *Reporter) log(sev codescan.Severity, msg string, attrs ...any) {
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
func (r *Reporter) relative(filename string) string {
	if filename == "" || r.Root == "" {
		return filename
	}

	rel, err := filepath.Rel(r.Root, filename)
	if err != nil {
		return filename
	}

	return rel
}
