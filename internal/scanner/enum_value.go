// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// enumLiteralValue converts the RHS expression of a `const Foo Kind = <expr>` declaration into its
// runtime value, reporting whether the expression is a literal this scanner understands.
//
// Go's scanner never produces a negative numeric literal: `-1` is a unary minus APPLIED to the
// literal `1`, so it reaches the AST as *ast.UnaryExpr wrapping *ast.BasicLit (same for the
// explicit `+1` spelling). Signed numeric literals are therefore unwrapped here — otherwise a
// `const PanLeft PanDirection = -1` would be dropped from the enum (go-swagger#3412).
//
// Anything else (identifiers — including iota-derived constants — calls, arithmetic) is reported as
// unsupported, and the caller skips that const. So is a numeric literal whose text does not parse,
// so that a const is never emitted as a nil enum member.
func enumLiteralValue(expr ast.Expr) (any, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return enumBasicLitValue(e)

	case *ast.UnaryExpr:
		if e.Op != token.SUB && e.Op != token.ADD {
			return nil, false
		}

		basicLit, ok := e.X.(*ast.BasicLit)
		if !ok {
			return nil, false
		}

		// A sign is only meaningful on a number: `-'a'` is legal Go but not a meaningful enum entry.
		if basicLit.Kind != token.INT && basicLit.Kind != token.FLOAT {
			return nil, false
		}

		sign := ""
		if e.Op == token.SUB {
			sign = "-"
		}

		return enumSignedLitValue(basicLit, sign)

	default:
		return nil, false
	}
}

// enumBasicLitValue converts the RHS of a `const Foo Kind = "bar"` declaration into its runtime
// value — int64 / float64 / unquoted string — for emission as an enum entry on the Swagger
// schema the scanner is building.
//
// Reports false when the literal kind is INT or FLOAT but the textual value fails to parse, so that
// the caller skips the const rather than emitting a nil enum member.
func enumBasicLitValue(basicLit *ast.BasicLit) (any, bool) {
	return enumSignedLitValue(basicLit, "")
}

// enumSignedLitValue is enumBasicLitValue with the sign carried by an enclosing unary operator
// folded back into the literal text.
//
// The sign is prepended rather than applied after parsing so that the whole int64 range round-trips:
// `-9223372036854775808` parses, whereas parsing `9223372036854775808` alone would overflow.
//
// Integers are parsed with base 0 so that the literal is read exactly as Go's own scanner wrote it:
// that accepts the `0x` / `0b` / `0o` prefixes, the legacy `0` octal form, and `_` digit
// separators. Base 10 would reject every one of those and, before the sign was handled here, quietly
// turned `017` into 17 where Go means 15.
func enumSignedLitValue(basicLit *ast.BasicLit, sign string) (any, bool) {
	switch basicLit.Kind.String() {
	case "INT":
		if result, err := strconv.ParseInt(sign+basicLit.Value, 0, 64); err == nil {
			return result, true
		}
	case "FLOAT":
		if result, err := strconv.ParseFloat(sign+basicLit.Value, 64); err == nil {
			return result, true
		}
	default:
		return strings.Trim(basicLit.Value, "\""), true
	}

	return nil, false
}
