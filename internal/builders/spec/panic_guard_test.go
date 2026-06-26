// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"go/token"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestBuilder_guard_RecoversPanicWithLocatedDiagnostic locks the §8.1 behaviour (go-swagger#2886):
// a panic in a per-declaration build step is recovered, surfaced as a located scan.internal-panic
// diagnostic naming the offending source (file:line + what), and converted into an aborting error
// wrapping ErrInternalPanic — never a raw Go stack trace.
//
// A non-panicking step is transparent.
// The spec.Builder build loops wrap every per-decl step in this guard.
func TestBuilder_guard_RecoversPanicWithLocatedDiagnostic(t *testing.T) {
	var diags []grammar.Diagnostic
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages:     []string{"./enhancements/emit-x-go-type/..."},
		WorkDir:      "../../../fixtures",
		OnDiagnostic: func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	b := NewBuilder(nil, ctx, false)

	pos := token.Position{Filename: "widget.go", Line: 42, Column: 7}

	gerr := b.guard(pos, "model Widget", func() error { panic("boom") })
	require.Error(t, gerr)
	require.ErrorIs(t, gerr, ErrInternalPanic)
	assert.Contains(t, gerr.Error(), "widget.go:42", "the aborting error carries the source location")
	assert.Contains(t, gerr.Error(), "model Widget")

	require.Len(t, diags, 1)
	assert.Equal(t, grammar.CodeInternalPanic, diags[0].Code)
	assert.Equal(t, grammar.SeverityError, diags[0].Severity)
	assert.Equal(t, pos, diags[0].Pos, "the diagnostic is anchored at the offending decl")
	assert.Contains(t, diags[0].Message, "boom")

	// A non-panicking step passes through with no diagnostic.
	require.NoError(t, b.guard(pos, "ok", func() error { return nil }))
	assert.Len(t, diags, 1, "a clean step emits no diagnostic")
}
