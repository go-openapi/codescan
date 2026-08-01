// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"go/constant"
	"go/token"
	"math"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestEnumConstantValue(t *testing.T) {
	t.Parallel()

	hugeInt := constant.BinaryOp(constant.MakeUint64(math.MaxUint64), token.MUL, constant.MakeInt64(2))

	for _, tc := range []struct {
		name       string
		value      constant.Value
		expected   any
		expectedOK bool
	}{
		{name: "int64", value: constant.MakeInt64(-1), expected: int64(-1), expectedOK: true},
		{
			name:     "int above MaxInt64 falls back to unsigned",
			value:    constant.MakeUint64(math.MaxUint64),
			expected: uint64(math.MaxUint64), expectedOK: true,
		},
		{name: "int beyond uint64 is dropped", value: hugeInt, expectedOK: false},
		{name: "float", value: constant.MakeFloat64(-0.5), expected: -0.5, expectedOK: true},
		{
			name:     "float needing rounding keeps the rounded value",
			value:    constant.BinaryOp(constant.MakeInt64(1), token.QUO, constant.MakeInt64(3)),
			expected: 1.0 / 3.0, expectedOK: true,
		},
		{name: "string", value: constant.MakeString("a\tb"), expected: "a\tb", expectedOK: true},
		{name: "bool", value: constant.MakeBool(true), expected: true, expectedOK: true},
		{name: "complex has no JSON representation", value: constant.ToComplex(constant.MakeInt64(1)), expectedOK: false},
		{name: "unknown (evaluation failed)", value: constant.MakeUnknown(), expectedOK: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			value, ok := enumConstantValue(tc.value)
			assert.Equal(t, tc.expectedOK, ok)
			if !tc.expectedOK {
				assert.Nil(t, value)

				return
			}
			assert.Equal(t, tc.expected, value)
		})
	}
}

// TestEnumLiteralValue covers the degraded reading, used only when the type-checker has no value
// for a constant. The literal forms it must still get right are the ones whose delimiters are not
// part of the value.
func TestEnumLiteralValue(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		kind     token.Token
		value    string
		expected any
	}{
		{name: "decimal int", kind: token.INT, value: "42", expected: int64(42)},
		{name: "hex int", kind: token.INT, value: "0x2a", expected: int64(42)},
		{name: "int above MaxInt64", kind: token.INT, value: "18446744073709551615", expected: uint64(math.MaxUint64)},
		{name: "float", kind: token.FLOAT, value: "-0.5", expected: -0.5},
		{name: "rune", kind: token.CHAR, value: `'a'`, expected: int64('a')},
		{name: "escaped rune", kind: token.CHAR, value: `'\t'`, expected: int64('\t')},
		{name: "quoted rune", kind: token.CHAR, value: `'\''`, expected: int64('\'')},
		{name: "interpreted string", kind: token.STRING, value: `"low"`, expected: "low"},
		{name: "escaped string", kind: token.STRING, value: `"a\tb"`, expected: "a\tb"},
		{name: "raw string", kind: token.STRING, value: "`raw`", expected: "raw"},
		{name: "unparsable int", kind: token.INT, value: "not-a-number", expected: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, enumLiteralValue(tc.kind, tc.value))
		})
	}
}
