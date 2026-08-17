// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// A finding is composed by the validator out of the document's own strings - path templates, parameter and property
// names, offending values - and every one of those came out of an annotation somebody wrote. So both halves of a row
// are scanned text, and neither may reach the terminal as a command.
const (
	rawSGR     = "\x1b[31mPWNED"
	encodedSGR = "␛[31mPWNED"
)

// assertEncoded is the shared check, written against a marker no style this package emits produces after an ESC.
func assertEncoded(t *testing.T, out string) {
	t.Helper()

	assert.NotContains(t, out, rawSGR, "a control sequence from the scanned source reached the terminal")
	assert.Contains(t, out, encodedSGR, "the control sequence was dropped instead of being shown")
}

func TestRender_EncodesControlSequences(t *testing.T) {
	findings := []Finding{{
		Severity: grammar.SeverityError,
		Pointer:  "/paths/~1pets/get/parameters/" + rawSGR,
		Message:  `path param "` + rawSGR + `" is not present in path "/pets"`,
	}}

	t.Run("styled row", func(t *testing.T) {
		// selected is out of range, so the only row rendered is the styled one.
		body, _ := Render(findings, true, nil, -1, true)
		assertEncoded(t, body)
	})

	t.Run("selected row", func(t *testing.T) {
		body, _ := Render(findings, true, nil, 0, true)
		assertEncoded(t, body)
	})

	t.Run("follower row", func(t *testing.T) {
		body, _ := Render(findings, true, nil, 0, false)
		assertEncoded(t, body)
	})
}

// TestRender_EncodesTheRunError covers the other way scanned text arrives: the validator refusing the document.
func TestRender_EncodesTheRunError(t *testing.T) {
	runErr := fmt.Errorf("%w: %s", errValidatorBroke, rawSGR)

	body, line := Render(nil, true, runErr, 0, true)

	assert.Equal(t, -1, line)
	assertEncoded(t, body)
}

// TestRender_KeepsAMessageOnItsOwnRows guards the block form.
//
// Render counts rows as it builds the body to report where the cursor sits, so a message that runs its own lines
// together would move the count off the row the reader is looking at.
func TestRender_KeepsAMessageOnItsOwnRows(t *testing.T) {
	findings := []Finding{{
		Severity: grammar.SeverityError,
		Pointer:  "/info",
		Message:  "first line\nsecond line",
	}}

	body, _ := Render(findings, true, nil, -1, true)

	assert.Contains(t, body, "first line")
	assert.Contains(t, body, "second line")
	// The tally's own break, plus the one inside the message.
	assert.Equal(t, 2, strings.Count(body, "\n"))
}
