// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"cmp"
	"go/token"
	"slices"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestBuilder_guard_RecoversPanicWithLocatedDiagnostic locks the behaviour raised by go-swagger#2886.
//
// A panic in a per-declaration build step is recovered, surfaced as a located scan.internal-panic diagnostic
// naming the offending source (file:line + what), and converted into an aborting error wrapping ErrInternalPanic
// — never a raw Go stack trace.
//
// A non-panicking step is transparent.
// The spec.Builder build loops wrap every per-decl step in this guard.
func TestBuilder_guard_RecoversPanicWithLocatedDiagnostic(t *testing.T) {
	var diags []grammar.Diagnostic
	ctx, err := scanner.NewScanCtx(applyLoader(&scanner.Options{
		Packages:     []string{"./enhancements/emit-x-go-type/..."},
		WorkDir:      "../../../testdata",
		OnDiagnostic: func(d grammar.Diagnostic) { diags = append(diags, d) },
	}))
	require.NoError(t, err)
	b := NewBuilder(nil, ctx, false)

	pos := token.Position{Filename: "widget.go", Line: 42, Column: 7}

	gerr := b.guard(pos, "model Widget", func() error { panic("boom") })
	require.Error(t, gerr)
	require.ErrorIs(t, gerr, ErrInternalPanic)
	assert.Contains(t, gerr.Error(), "widget.go:42", "the aborting error carries the source location")
	assert.Contains(t, gerr.Error(), "model Widget")

	require.NotEmpty(t, diags)
	diagsAtThisPoint := len(diags)
	mostSevere := slices.MaxFunc(diags, func(a, b grammar.Diagnostic) int { // depending on the loader mode, some hint diagnostics may be added
		return cmp.Compare(int(b.Severity), int(a.Severity))
	})
	assert.Equal(t, grammar.CodeInternalPanic, mostSevere.Code)
	assert.Equal(t, grammar.SeverityError, mostSevere.Severity)
	assert.Equal(t, pos, mostSevere.Pos, "the diagnostic is anchored at the offending decl")
	assert.Contains(t, mostSevere.Message, "boom")

	// A non-panicking step passes through with no additional diagnostic.
	require.NoError(t, b.guard(pos, "ok", func() error { return nil }))
	assert.Len(t, diags, diagsAtThisPoint, "a clean step emits no additional diagnostic")
}

func applyLoader(opts *scanner.Options) *scanner.Options {
	return scantest.ApplyLoader(opts)
}
