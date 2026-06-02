// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package validations_test

import (
	"testing"

	"github.com/go-openapi/codescan/internal/builders/validations"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestCoerceValue_NilSchemaReturnsRaw(t *testing.T) {
	got, err := validations.CoerceValue("hello", nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestCoerceValue_PrimitiveTypes(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		schema  *spec.SimpleSchema
		want    any
		wantErr bool
	}{
		{"integer", "42", &spec.SimpleSchema{Type: "integer"}, 42, false},
		{"int64", "100", &spec.SimpleSchema{Type: "int64"}, 100, false},
		{"int32", "10", &spec.SimpleSchema{Type: "int32"}, 10, false},
		{"boolean true", "true", &spec.SimpleSchema{Type: "boolean"}, true, false},
		{"boolean false", "false", &spec.SimpleSchema{Type: "bool"}, false, false},
		{"number", "1.5", &spec.SimpleSchema{Type: "number"}, 1.5, false},
		{"float64", "3.14", &spec.SimpleSchema{Type: "float64"}, 3.14, false},
		{"unknown type returns raw", "anything", &spec.SimpleSchema{Type: "weird"}, "anything", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validations.CoerceValue(tc.raw, tc.schema)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCoerceValue_ObjectAndArrayUnmarshal(t *testing.T) {
	obj, err := validations.CoerceValue(`{"k":1}`, &spec.SimpleSchema{Type: "object"})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"k": float64(1)}, obj)

	arr, err := validations.CoerceValue(`[1,2,3]`, &spec.SimpleSchema{Type: "array"})
	require.NoError(t, err)
	assert.Equal(t, []any{float64(1), float64(2), float64(3)}, arr)
}

func TestCoerceValue_InvalidJSONFallsBackToRaw(t *testing.T) {
	// v1 quirk preserved: object/array parse failure returns raw,
	// not error.
	got, err := validations.CoerceValue("not json", &spec.SimpleSchema{Type: "object"})
	require.NoError(t, err)
	assert.Equal(t, "not json", got)
}

func TestCoerceValue_FormatPrecedenceQuirk(t *testing.T) {
	// v1 quirk: SimpleSchema.TypeName() returns Format when Format is
	// non-empty. With Type="number" and Format="float", TypeName()
	// returns "float" — NOT in the switch — so the value falls
	// through to default and is returned as raw string. The schema
	// builder's schemeFromPS deliberately drops Format to avoid this.
	// This test pins the v1-bug behaviour for parity through S1–S6;
	// S3 introduces a corrected path.
	got, err := validations.CoerceValue("1.5", &spec.SimpleSchema{Type: "number", Format: "float"})
	require.NoError(t, err)
	assert.Equal(t, "1.5", got, "v1 Format-wins quirk: raw string returned for unrecognized 'float'")
}

func TestCoerceEnum_JSONArrayForm(t *testing.T) {
	out := validations.CoerceEnum(`["low","medium","high"]`, &spec.SimpleSchema{Type: "string"})
	assert.Equal(t, []any{"low", "medium", "high"}, out)
}

func TestCoerceEnum_CommaListForm(t *testing.T) {
	out := validations.CoerceEnum(`a, b, c`, &spec.SimpleSchema{Type: "string"})
	assert.Equal(t, []any{"a", "b", "c"}, out)
}

func TestCoerceEnum_PerItemTyping(t *testing.T) {
	out := validations.CoerceEnum(`[1, 2, 3]`, &spec.SimpleSchema{Type: "integer"})
	assert.Equal(t, []any{1, 2, 3}, out)
}

func TestCoerceEnum_CommaListWhitespaceTrimmed(t *testing.T) {
	out := validations.CoerceEnum(`  a  ,   b  ,c  `, &spec.SimpleSchema{Type: "string"})
	assert.Equal(t, []any{"a", "b", "c"}, out)
}
