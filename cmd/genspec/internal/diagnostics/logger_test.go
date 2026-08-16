// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestLogger(t *testing.T) {
	t.Parallel()

	t.Run("should write the message at the level it was given", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		logger(&out, false, false).Warn("something to say")

		assert.Contains(t, out.String(), "something to say")
		assert.Contains(t, out.String(), "WARN")
		assert.NotContains(t, out.String(), ":", "no timestamp: it is noise in a command that runs once")
	})

	t.Run("should leave no escape codes behind when colour is refused", func(t *testing.T) {
		// Only this half is checkable here. Asking FOR colour cannot be observed from a test, because the colour
		// library decides globally, at init, from whether os.Stdout is a terminal - which under `go test` it is not.
		t.Parallel()

		var out bytes.Buffer
		logger(&out, false, false).Error("a real problem")

		assert.NotContains(t, out.String(), "\x1b[")
	})

	t.Run("should say nothing without a stream to say it to", func(t *testing.T) {
		// A nil stream is a caller asking for no diagnostics at all, not one asking to write to nowhere:
		// the difference is that nothing is formatted.
		t.Parallel()

		require.NotPanics(t, func() { logger(nil, true, false).Error("into the void") })
	})
}

// The handler -quiet installs refuses every level, so slog never asks it to format anything. It still has to answer
// the rest of the interface: a caller is free to decorate a logger it is about to find has nothing to say.
func TestDiscardHandler(t *testing.T) {
	t.Parallel()

	t.Run("should refuse every level", func(t *testing.T) {
		t.Parallel()

		for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
			assert.Falsef(t, discardHandler{}.Enabled(context.Background(), level),
				"level %s is enabled, so a diagnostic would be formatted before being dropped", level)
		}
	})

	t.Run("should stay itself when decorated", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		quiet := slog.New(discardHandler{}).With(slog.String("scan", "one")).WithGroup("findings")

		require.NotPanics(t, func() { quiet.Error("a real problem") })
		assert.Empty(t, out.String())
	})

	t.Run("should drop a record it is handed anyway", func(t *testing.T) {
		// Reachable by anything holding the handler rather than the logger, which is why it answers at all.
		t.Parallel()

		require.NoError(t, discardHandler{}.Handle(context.Background(), slog.Record{}))
	})
}
