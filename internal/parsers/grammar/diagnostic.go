// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"fmt"
	"go/token"
)

// Severity classifies a Diagnostic's seriousness. The parser never
// aborts; callers (analyzers, LSP, the CLI) decide policy based on
// severity.
type Severity int

const (
	SeverityError   Severity = iota // parse failed to interpret the line at the reported position
	SeverityWarning                 // parse succeeded but something looks wrong (e.g., deprecated alias)
	SeverityHint                    // informational, e.g., suggest a canonical spelling
)

// String renders a Severity for logs and CLI output.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityHint:
		return "hint"
	default:
		return fmt.Sprintf("severity(%d)", int(s))
	}
}

// Code is a stable identifier for a class of Diagnostic. LSP clients
// filter and group by Code; tests assert on Code rather than Message.
//
// Codes are dotted, lowercase, and prefixed with "parse." for parser-side
// diagnostics. Downstream sub-parsers (e.g., internal/parsers/yaml/) use
// their own prefix (e.g., "yaml.") so codes remain globally unique.
type Code string

// Known diagnostic codes emitted by the grammar parser. This list grows
// as P1–P2 land; every new emission site picks an existing code or adds
// a constant here so the set stays discoverable.
const (
	CodeInvalidNumber     Code = "parse.invalid-number"
	CodeInvalidInteger    Code = "parse.invalid-integer"
	CodeInvalidBoolean    Code = "parse.invalid-boolean"
	CodeInvalidStringEnum Code = "parse.invalid-string-enum"
	CodeUnknownKeyword    Code = "parse.unknown-keyword"
	CodeContextInvalid    Code = "parse.context-invalid"
	CodeInvalidExtension  Code = "parse.invalid-extension-name"
	CodeUnterminatedYAML  Code = "parse.unterminated-yaml"
	CodeInvalidAnnotation Code = "parse.invalid-annotation"
	CodeMalformedLine     Code = "parse.malformed-line"
)

// Diagnostic is one observation about a comment block. The parser
// accumulates Diagnostics into a slice on the enclosing Block (see
// ast.go) rather than throwing; callers decide whether a given severity
// is fatal for their flow.
type Diagnostic struct {
	Pos      token.Position
	Severity Severity
	Code     Code
	Message  string
}

// String renders a Diagnostic in a compiler-style, one-line form:
//
//	file:line:col: severity: message [code]
//
// Keep the format parseable by editor jump-to-line tooling.
func (d Diagnostic) String() string {
	loc := d.Pos.String()
	if loc == "-" || loc == "" {
		loc = "<unknown>"
	}
	return fmt.Sprintf("%s: %s: %s [%s]", loc, d.Severity, d.Message, d.Code)
}

// Errorf builds a SeverityError Diagnostic with a formatted message.
func Errorf(pos token.Position, code Code, format string, args ...any) Diagnostic {
	return Diagnostic{Pos: pos, Severity: SeverityError, Code: code, Message: fmt.Sprintf(format, args...)}
}

// Warnf builds a SeverityWarning Diagnostic with a formatted message.
func Warnf(pos token.Position, code Code, format string, args ...any) Diagnostic {
	return Diagnostic{Pos: pos, Severity: SeverityWarning, Code: code, Message: fmt.Sprintf(format, args...)}
}

// Hintf builds a SeverityHint Diagnostic with a formatted message.
func Hintf(pos token.Position, code Code, format string, args ...any) Diagnostic {
	return Diagnostic{Pos: pos, Severity: SeverityHint, Code: code, Message: fmt.Sprintf(format, args...)}
}
