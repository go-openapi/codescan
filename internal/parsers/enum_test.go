// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/spec"
)

func Test_getEnumBasicLitValue(t *testing.T) {
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.INT, Value: "0"}, int64(0))
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.INT, Value: "-1"}, int64(-1))
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.INT, Value: "42"}, int64(42))
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.INT, Value: ""}, nil)
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.INT, Value: "word"}, nil)

	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.FLOAT, Value: "0"}, float64(0))
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.FLOAT, Value: "-1"}, float64(-1))
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.FLOAT, Value: "42"}, float64(42))
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.FLOAT, Value: "1.1234"}, float64(1.1234))
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.FLOAT, Value: "1.9876"}, float64(1.9876))
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.FLOAT, Value: ""}, nil)
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.FLOAT, Value: "word"}, nil)

	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.STRING, Value: "Foo"}, "Foo")
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.STRING, Value: ""}, "")
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.STRING, Value: "0"}, "0")
	verifyGetEnumBasicLitValue(t, ast.BasicLit{Kind: token.STRING, Value: "1.1"}, "1.1")
}

func verifyGetEnumBasicLitValue(t *testing.T, basicLit ast.BasicLit, expected any) {
	actual := GetEnumBasicLitValue(&basicLit)

	assert.Equal(t, expected, actual)
}

func TestParseValueFromSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		schema *spec.SimpleSchema
		want   any
	}{
		{"nil schema", "hello", nil, "hello"},
		{"string", "hello", &spec.SimpleSchema{Type: "string"}, "hello"},
		{"integer", "42", &spec.SimpleSchema{Type: "integer"}, 42},
		{"int64", "100", &spec.SimpleSchema{Type: "int64"}, 100},
		{"bool true", "true", &spec.SimpleSchema{Type: "bool"}, true},
		{"boolean false", "false", &spec.SimpleSchema{Type: "boolean"}, false},
		{"float64", "3.14", &spec.SimpleSchema{Type: "float64"}, float64(3.14)},
		{"number", "2.5", &spec.SimpleSchema{Type: "number"}, float64(2.5)},
		{"object valid", `{"a":"b"}`, &spec.SimpleSchema{Type: "object"}, map[string]any{"a": "b"}},
		{"object invalid json", `not-json`, &spec.SimpleSchema{Type: "object"}, "not-json"},
		{"array valid", `[1,2,3]`, &spec.SimpleSchema{Type: "array"}, []any{float64(1), float64(2), float64(3)}},
		{"array invalid json", `not-json`, &spec.SimpleSchema{Type: "array"}, "not-json"},
		{"unknown type", "raw", &spec.SimpleSchema{Type: "custom"}, "raw"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseValueFromSchema(tc.input, tc.schema)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("integer parse error", func(t *testing.T) {
		_, err := parseValueFromSchema("not-a-number", &spec.SimpleSchema{Type: "integer"})
		require.Error(t, err)
	})

	t.Run("bool parse error", func(t *testing.T) {
		_, err := parseValueFromSchema("maybe", &spec.SimpleSchema{Type: "bool"})
		require.Error(t, err)
	})
}

func TestParseEnum(t *testing.T) {
	t.Parallel()

	t.Run("JSON format strings", func(t *testing.T) {
		result := ParseEnum(`["a","b","c"]`, &spec.SimpleSchema{Type: "string"})
		assert.Equal(t, []any{"a", "b", "c"}, result)
	})

	t.Run("JSON format integers", func(t *testing.T) {
		result := ParseEnum(`[1,2,3]`, &spec.SimpleSchema{Type: "integer"})
		assert.Equal(t, []any{1, 2, 3}, result)
	})

	t.Run("old comma-separated format", func(t *testing.T) {
		result := ParseEnum("a,b,c", &spec.SimpleSchema{Type: "string"})
		assert.Equal(t, []any{"a", "b", "c"}, result)
	})

	t.Run("old format integers", func(t *testing.T) {
		result := ParseEnum("1,2,3", &spec.SimpleSchema{Type: "integer"})
		assert.Equal(t, []any{1, 2, 3}, result)
	})

	t.Run("old format with parse error fallback", func(t *testing.T) {
		// "abc" cannot be parsed as integer → fallback to raw string
		result := ParseEnum("abc,2,xyz", &spec.SimpleSchema{Type: "integer"})
		assert.Equal(t, []any{"abc", 2, "xyz"}, result)
	})

	t.Run("JSON format with parse error fallback", func(t *testing.T) {
		// JSON array of integers, but "abc" can't parse as integer → fallback
		result := ParseEnum(`["abc",2,"xyz"]`, &spec.SimpleSchema{Type: "integer"})
		assert.Equal(t, []any{"abc", 2, "xyz"}, result)
	})
}

func TestGetEnumDesc(t *testing.T) {
	t.Parallel()

	t.Run("with extension", func(t *testing.T) {
		ext := spec.Extensions{"x-go-enum-desc": "Active - active state\nInactive - inactive state"}
		assert.EqualT(t, "Active - active state\nInactive - inactive state", GetEnumDesc(ext))
	})

	t.Run("without extension", func(t *testing.T) {
		ext := spec.Extensions{}
		assert.EqualT(t, "", GetEnumDesc(ext))
	})

	t.Run("nil extensions", func(t *testing.T) {
		assert.EqualT(t, "", GetEnumDesc(nil))
	})
}

func TestEnumDescExtension(t *testing.T) {
	t.Parallel()

	assert.EqualT(t, "x-go-enum-desc", EnumDescExtension())
}
