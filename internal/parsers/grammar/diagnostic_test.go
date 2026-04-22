// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/token"
	"strings"
	"testing"
)

func TestSeverityString(t *testing.T) {
	cases := []struct {
		in   Severity
		want string
	}{
		{SeverityError, "error"},
		{SeverityWarning, "warning"},
		{SeverityHint, "hint"},
		{Severity(99), "severity(99)"},
	}
	for _, tc := range cases {
		if got := tc.in.String(); got != tc.want {
			t.Errorf("Severity(%d).String() = %q want %q", int(tc.in), got, tc.want)
		}
	}
}

func TestDiagnosticConstructors(t *testing.T) {
	pos := token.Position{Filename: "foo.go", Line: 12, Column: 3}

	err := Errorf(pos, CodeInvalidNumber, "bad %s", "value")
	if err.Severity != SeverityError || err.Code != CodeInvalidNumber || err.Message != "bad value" {
		t.Errorf("Errorf built unexpected Diagnostic: %+v", err)
	}

	warn := Warnf(pos, CodeContextInvalid, "context mismatch")
	if warn.Severity != SeverityWarning || warn.Code != CodeContextInvalid {
		t.Errorf("Warnf built unexpected Diagnostic: %+v", warn)
	}

	hint := Hintf(pos, CodeUnknownKeyword, "did you mean %q?", "maximum")
	if hint.Severity != SeverityHint || hint.Message != `did you mean "maximum"?` {
		t.Errorf("Hintf built unexpected Diagnostic: %+v", hint)
	}
}

func TestDiagnosticString(t *testing.T) {
	pos := token.Position{Filename: "foo.go", Line: 12, Column: 3}
	d := Errorf(pos, CodeInvalidNumber, "bad value")
	got := d.String()
	want := "foo.go:12:3: error: bad value [parse.invalid-number]"
	if got != want {
		t.Errorf("Diagnostic.String()\n got: %s\nwant: %s", got, want)
	}

	// Position-less diagnostic should still render.
	d2 := Errorf(token.Position{}, CodeInvalidNumber, "bad value")
	if !strings.Contains(d2.String(), "<unknown>") {
		t.Errorf("empty position should render as <unknown>: %s", d2.String())
	}
}
