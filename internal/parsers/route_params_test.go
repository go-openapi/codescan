// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

func TestSetOpParams_Matches(t *testing.T) {
	t.Parallel()

	sp := NewSetParams(nil, nil)
	assert.TrueT(t, sp.Matches("parameters:"))
	assert.TrueT(t, sp.Matches("Parameters:"))
	assert.FalseT(t, sp.Matches("something else"))
}

func TestSetOpParams_Parse(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		sp := NewSetParams(nil, func(_ []*oaispec.Parameter) {})
		require.NoError(t, sp.Parse(nil))
		require.NoError(t, sp.Parse([]string{}))
		require.NoError(t, sp.Parse([]string{""}))
	})

	t.Run("single param", func(t *testing.T) {
		var got []*oaispec.Parameter
		sp := NewSetParams(nil, func(params []*oaispec.Parameter) { got = params })

		lines := []string{
			"+ name: id",
			"  in: path",
			"  type: integer",
			"  required: true",
			"  description: The pet ID",
		}
		require.NoError(t, sp.Parse(lines))
		require.Len(t, got, 1)
		assert.EqualT(t, "id", got[0].Name)
		assert.EqualT(t, "path", got[0].In)
		assert.TrueT(t, got[0].Required)
		assert.EqualT(t, "The pet ID", got[0].Description)
		assert.EqualT(t, "integer", got[0].Type)
	})

	t.Run("multiple params", func(t *testing.T) {
		var got []*oaispec.Parameter
		sp := NewSetParams(nil, func(params []*oaispec.Parameter) { got = params })

		lines := []string{
			"+ name: limit",
			"  in: query",
			"  type: integer",
			"+ name: offset",
			"  in: query",
			"  type: integer",
		}
		require.NoError(t, sp.Parse(lines))
		require.Len(t, got, 2)
		assert.EqualT(t, "limit", got[0].Name)
		assert.EqualT(t, "offset", got[1].Name)
	})

	t.Run("body param", func(t *testing.T) {
		var got []*oaispec.Parameter
		sp := NewSetParams(nil, func(params []*oaispec.Parameter) { got = params })

		lines := []string{
			"+ name: body",
			"  in: body",
			"  type: Pet",
		}
		require.NoError(t, sp.Parse(lines))
		require.Len(t, got, 1)
		assert.EqualT(t, "body", got[0].In)
		assert.EqualT(t, "", got[0].Type) // body params clear type
		require.NotNil(t, got[0].Schema)
	})

	t.Run("boolean type", func(t *testing.T) {
		var got []*oaispec.Parameter
		sp := NewSetParams(nil, func(params []*oaispec.Parameter) { got = params })

		lines := []string{
			"+ name: active",
			"  in: query",
			"  type: bool",
		}
		require.NoError(t, sp.Parse(lines))
		require.Len(t, got, 1)
		assert.EqualT(t, "boolean", got[0].Type) // "bool" → "boolean"
	})

	t.Run("allow empty value", func(t *testing.T) {
		var got []*oaispec.Parameter
		sp := NewSetParams(nil, func(params []*oaispec.Parameter) { got = params })

		lines := []string{
			"+ name: q",
			"  in: query",
			"  type: string",
			"  allowempty: true",
		}
		require.NoError(t, sp.Parse(lines))
		require.Len(t, got, 1)
		assert.TrueT(t, got[0].AllowEmptyValue)
	})

	t.Run("error no leading +", func(t *testing.T) {
		sp := NewSetParams(nil, func(_ []*oaispec.Parameter) {})
		err := sp.Parse([]string{"name: id"})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrParser)
	})

	t.Run("with schema extras", func(t *testing.T) {
		var got []*oaispec.Parameter
		sp := NewSetParams(nil, func(params []*oaispec.Parameter) { got = params })

		lines := []string{
			"+ name: age",
			"  in: query",
			"  type: integer",
			"  min: 0",
			"  max: 150",
			"  default: 25",
			"  enum: 18,25,30,65",
			"  format: int32",
		}
		require.NoError(t, sp.Parse(lines))
		require.Len(t, got, 1)
		assert.EqualT(t, "int32", got[0].Format)
		assert.Equal(t, float64(25), got[0].Default)
	})
}

func TestConvert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typeStr  string
		valueStr string
		want     any
	}{
		{"integer", "42", float64(42)},
		{"number", "3.14", float64(3.14)},
		{"boolean", "true", true},
		{"bool", "false", false},
		{"string", "hello", "hello"},
		{"integer", "not-a-number", "not-a-number"},
		{"boolean", "maybe", "maybe"},
		{"", "raw", "raw"},
	}

	for _, tc := range tests {
		t.Run(tc.typeStr+"/"+tc.valueStr, func(t *testing.T) {
			assert.Equal(t, tc.want, convert(tc.typeStr, tc.valueStr))
		})
	}
}

func TestGetType(t *testing.T) {
	t.Parallel()

	t.Run("empty type", func(t *testing.T) {
		s := &oaispec.Schema{}
		assert.EqualT(t, "", getType(s))
	})

	t.Run("with type", func(t *testing.T) {
		s := &oaispec.Schema{}
		s.Type = oaispec.StringOrArray{"string"}
		assert.EqualT(t, "string", getType(s))
	})
}

func TestSetOpParams_ParseLineWithoutColon(t *testing.T) {
	t.Parallel()

	var got []*oaispec.Parameter
	sp := NewSetParams(nil, func(params []*oaispec.Parameter) { got = params })
	// A line after "+" that has no colon is silently skipped
	lines := []string{
		"+ name: test",
		"  no-colon-here",
		"  in: query",
		"  type: string",
	}
	require.NoError(t, sp.Parse(lines))
	require.Len(t, got, 1)
	assert.EqualT(t, "test", got[0].Name)
}

func TestSetOpParams_ArraySchemaExtras(t *testing.T) {
	t.Parallel()

	var got []*oaispec.Parameter
	sp := NewSetParams(nil, func(params []*oaispec.Parameter) { got = params })
	lines := []string{
		"+ name: tags",
		"  in: query",
		"  type: array",
		"  minlength: 1",
		"  maxlength: 10",
		"  enum: a,b,c",
		"  description: A list of tags",
	}
	require.NoError(t, sp.Parse(lines))
	require.Len(t, got, 1)
	// For non-body params, schema is converted to validations
	assert.EqualT(t, "array", got[0].Type)
}

func TestSetOpParams_NilSchemaGuard(t *testing.T) {
	t.Parallel()

	// A param with no type: field won't have a schema
	var got []*oaispec.Parameter
	sp := NewSetParams(nil, func(params []*oaispec.Parameter) { got = params })
	lines := []string{
		"+ name: simple",
		"  in: query",
	}
	require.NoError(t, sp.Parse(lines))
	require.Len(t, got, 1)
	assert.Nil(t, got[0].Schema)
}

func TestSetOpParams_InvalidIn(t *testing.T) {
	t.Parallel()

	var got []*oaispec.Parameter
	sp := NewSetParams(nil, func(params []*oaispec.Parameter) { got = params })
	lines := []string{
		"+ name: test",
		"  in: cookie",
		"  type: string",
	}
	require.NoError(t, sp.Parse(lines))
	require.Len(t, got, 1)
	assert.EqualT(t, "", got[0].In) // "cookie" is not in validIn
}
