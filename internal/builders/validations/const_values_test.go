// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package validations_test

import (
	"math"
	"testing"

	"github.com/go-openapi/codescan/internal/builders/validations"
	"github.com/go-openapi/testify/v2/assert"
)

func TestCoerceConstant(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		value      any
		schemaType string
		expected   any
		expectedOK bool
	}{
		// integer targets
		{name: "int64 on integer", value: int64(-1), schemaType: "integer", expected: int64(-1), expectedOK: true},
		{name: "uint64 on integer", value: uint64(math.MaxUint64), schemaType: "integer", expected: uint64(math.MaxUint64), expectedOK: true},
		{name: "integral float on integer", value: 42.0, schemaType: "integer", expected: int64(42), expectedOK: true},
		{name: "negative integral float on integer", value: -42.0, schemaType: "integer", expected: int64(-42), expectedOK: true},
		{name: "fractional float on integer", value: 0.5, schemaType: "integer", expectedOK: false},
		{name: "out-of-range float on integer", value: 1e300, schemaType: "integer", expectedOK: false},
		{name: "MaxInt64+1 as float on integer", value: 9223372036854775808.0, schemaType: "integer", expectedOK: false},
		{name: "NaN on integer", value: math.NaN(), schemaType: "integer", expectedOK: false},
		{name: "string on integer", value: "nope", schemaType: "integer", expectedOK: false},

		// number targets
		{name: "float on number", value: -0.5, schemaType: "number", expected: -0.5, expectedOK: true},
		{name: "int64 on number", value: int64(0), schemaType: "number", expected: float64(0), expectedOK: true},
		{name: "uint64 on number", value: uint64(3), schemaType: "number", expected: float64(3), expectedOK: true},
		{name: "string on number", value: "nope", schemaType: "number", expectedOK: false},

		// string / boolean targets
		{name: "string on string", value: "low", schemaType: "string", expected: "low", expectedOK: true},
		{name: "int64 on string", value: int64(1), schemaType: "string", expectedOK: false},
		{name: "bool on boolean", value: true, schemaType: "boolean", expected: true, expectedOK: true},
		{name: "int64 on boolean", value: int64(1), schemaType: "boolean", expectedOK: false},

		// unresolved target: pass through untouched rather than guess
		{name: "int64 on unresolved type", value: int64(7), schemaType: "", expected: int64(7), expectedOK: true},
		{name: "string on object type", value: "raw", schemaType: "object", expected: "raw", expectedOK: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			value, ok := validations.CoerceConstant(tc.value, tc.schemaType)
			assert.Equal(t, tc.expectedOK, ok)
			if !tc.expectedOK {
				assert.Nil(t, value)

				return
			}
			assert.Equal(t, tc.expected, value)
		})
	}
}
