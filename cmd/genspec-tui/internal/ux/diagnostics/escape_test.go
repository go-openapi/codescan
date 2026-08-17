// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import (
	stderrors "errors"
	"fmt"
	"go/token"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// Both halves of a diagnostic row come from the scanned tree: the location names a file, and the message quotes the
// annotation that provoked it. Whether the scanner happens to have quoted what it interpolated is not something the
// pane can draw on faith.
const (
	rawSGR     = "\x1b[31mPWNED"
	encodedSGR = "␛[31mPWNED"
)

// errScanBroke stands in for codescan failing outright, as opposed to reporting diagnostics.
var errScanBroke = stderrors.New("could not scan source")

// assertEncoded is the shared check, written against a marker no style this package emits produces after an ESC.
func assertEncoded(t *testing.T, out string) {
	t.Helper()

	assert.NotContains(t, out, rawSGR, "a control sequence from the scanned tree reached the terminal")
	assert.Contains(t, out, encodedSGR, "the control sequence was dropped instead of being shown")
}

func TestRender_EncodesAMessage(t *testing.T) {
	diags := []grammar.Diagnostic{{
		Severity: grammar.SeverityError,
		Code:     grammar.CodeInvalidAnnotation,
		Message:  "unknown swagger annotation " + rawSGR,
		Pos:      token.Position{Filename: "/work/api.go", Line: 12, Column: 3},
	}}

	t.Run("styled row", func(t *testing.T) {
		body, _ := Render("/work", nil, diags, -1, true)
		assertEncoded(t, body)
	})

	t.Run("selected row", func(t *testing.T) {
		body, _ := Render("/work", nil, diags, 0, true)
		assertEncoded(t, body)
	})
}

// TestRender_EncodesTheLocation covers the file NAME, which a repository chooses.
func TestRender_EncodesTheLocation(t *testing.T) {
	diags := []grammar.Diagnostic{{
		Severity: grammar.SeverityWarning,
		Code:     grammar.CodeInvalidAnnotation,
		Message:  "something is off",
		Pos:      token.Position{Filename: "/work/" + rawSGR + ".go", Line: 1, Column: 1},
	}}

	t.Run("inside the work dir", func(t *testing.T) {
		body, _ := Render("/work", nil, diags, -1, true)
		assertEncoded(t, body)
	})

	t.Run("outside the work dir", func(t *testing.T) {
		body, _ := Render("/elsewhere", nil, diags, -1, true)
		assertEncoded(t, body)
	})
}

// TestRender_EncodesAScanError covers the hard failure shown above the list.
func TestRender_EncodesAScanError(t *testing.T) {
	body, _ := Render("/work", fmt.Errorf("%w: %s", errScanBroke, rawSGR), nil, -1, true)

	assertEncoded(t, body)
}

// TestRender_KeepsAMessageOnItsOwnRows guards the block form.
//
// Render counts rows as it builds the body so the pane can reveal the selected one, which a message run together with
// its own line breaks would put out of step.
func TestRender_KeepsAMessageOnItsOwnRows(t *testing.T) {
	diags := []grammar.Diagnostic{{
		Severity: grammar.SeverityHint,
		Code:     grammar.CodeInvalidAnnotation,
		Message:  "first line\nsecond line",
		Pos:      token.Position{Filename: "/work/api.go", Line: 1, Column: 1},
	}}

	body, _ := Render("/work", nil, diags, -1, true)

	assert.Contains(t, body, "first line")
	assert.Contains(t, body, "second line")
	// The tally's own break, plus the one inside the message.
	assert.Equal(t, 2, strings.Count(body, "\n"))
}
