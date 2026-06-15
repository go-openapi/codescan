// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"fmt"
	"go/token"
)

// Severity classifies a Diagnostic's seriousness. The parser never
// aborts; callers (analyzers, LSP, the CLI) decide policy by
// severity. See README §diagnostics.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityHint
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

// Code is a stable identifier for a class of Diagnostic.
type Code string

// Diagnostic codes. The `parse.*` prefix marks lexer/parser-level
// observations; `validate.*` marks semantic-validation observations
// emitted by the builder layer (typically through the
// internal/builders/validations package).
const (
	CodeInvalidNumber     Code = "parse.invalid-number"
	CodeInvalidInteger    Code = "parse.invalid-integer"
	CodeInvalidBoolean    Code = "parse.invalid-boolean"
	CodeInvalidEnumOption Code = "parse.invalid-enum-option"
	CodeContextInvalid    Code = "parse.context-invalid"
	CodeInvalidExtension  Code = "parse.invalid-extension-name"
	// CodeInvalidYAMLExtensions fires when the body of an
	// `extensions:` raw block fails YAML parsing. The block is
	// skipped (no Extension entries emitted) and a warning is raised.
	CodeInvalidYAMLExtensions Code = "parse.invalid-yaml-extensions"
	CodeUnterminatedYAML      Code = "parse.unterminated-yaml"
	CodeInvalidAnnotation     Code = "parse.invalid-annotation"
	CodeInvalidTypeRef        Code = "parse.invalid-type-ref"
	CodeUnexpectedToken       Code = "parse.unexpected-token"
	CodeMalformedOperation    Code = "parse.malformed-operation"
	CodeMissingRequiredArg    Code = "parse.missing-required-arg"

	// CodeShapeMismatch fires when a keyword is applied to a schema
	// whose resolved Swagger type doesn't match the keyword's domain
	// (e.g. `pattern: ^a$` on an integer field). Emitted by
	// internal/builders/validations.IsLegalForType callers.
	CodeShapeMismatch Code = "validate.shape-mismatch"

	// CodeAmbiguousEmbed fires when two embedded types of a parent
	// struct (or struct embed-chains at the same depth) both promote
	// a property with the same JSON name but different Go names. Go's
	// own rule is to not promote such ambiguous fields; codescan
	// currently emits a last-write-wins schema regardless. The
	// diagnostic surfaces the case so authors can disambiguate.
	CodeAmbiguousEmbed Code = "validate.ambiguous-embed"

	// CodeUnsupportedInSimpleSchema fires when the schema builder
	// running in SimpleSchema mode produces an outcome that OAS v2
	// does not allow on a parameter/header (object type, $ref, allOf,
	// properties, …). The diagnostic is emitted at exit and the
	// target is reset to empty `{}` — honest over lossy. Reaching
	// this code path typically means a non-body parameter or response
	// header was typed as a struct or interface that the recognizer
	// cascade couldn't reduce to a primitive.
	CodeUnsupportedInSimpleSchema Code = "validate.unsupported-in-simple-schema"

	// CodeUnsupportedType fires when a `swagger:type` argument cannot be
	// resolved to an inline schema: an unknown type name, a `file`
	// override (use swagger:file instead), or a keyword used where it is
	// not valid (e.g. `inline`/`array` as an array element). The override
	// is dropped and the subject falls through to its Go type. See the F3
	// reconciliation in .claude/plans/quirks-F-series-fix.md.
	CodeUnsupportedType Code = "validate.unsupported-type"

	// CodeDeprecated fires when an accepted-but-deprecated annotation or
	// keyword value is used (the input is still processed). Carries a
	// migration hint in the message — e.g. `swagger:type array` →
	// `inline`, or the deprecated `swagger:alias` annotation.
	CodeDeprecated Code = "validate.deprecated"
)

// Diagnostic is one observation about a comment block.
type Diagnostic struct {
	Pos      token.Position
	Severity Severity
	Code     Code
	Message  string
}

// String renders a Diagnostic in compiler-style one-line form.
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

// Warnf builds a SeverityWarning Diagnostic.
func Warnf(pos token.Position, code Code, format string, args ...any) Diagnostic {
	return Diagnostic{Pos: pos, Severity: SeverityWarning, Code: code, Message: fmt.Sprintf(format, args...)}
}

// Hintf builds a SeverityHint Diagnostic.
func Hintf(pos token.Position, code Code, format string, args ...any) Diagnostic {
	return Diagnostic{Pos: pos, Severity: SeverityHint, Code: code, Message: fmt.Sprintf(format, args...)}
}
