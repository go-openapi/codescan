// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan/internal/ifaces"
	"github.com/go-openapi/codescan/internal/scantest/mocks"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

// validationRecorder captures all calls made to a ValidationBuilder.
type validationRecorder struct {
	maximum          *float64
	exclusiveMaximum bool
	minimum          *float64
	exclusiveMinimum bool
	multipleOf       *float64
	minItems         *int64
	maxItems         *int64
	minLength        *int64
	maxLength        *int64
	pattern          string
	unique           *bool
	enum             string
	defaultVal       any
	exampleVal       any
	collectionFormat string
}

func (r *validationRecorder) SetMaximum(v float64, exclusive bool) {
	r.maximum = &v
	r.exclusiveMaximum = exclusive
}

func (r *validationRecorder) SetMinimum(v float64, exclusive bool) {
	r.minimum = &v
	r.exclusiveMinimum = exclusive
}
func (r *validationRecorder) SetMultipleOf(v float64)      { r.multipleOf = &v }
func (r *validationRecorder) SetMinItems(v int64)          { r.minItems = &v }
func (r *validationRecorder) SetMaxItems(v int64)          { r.maxItems = &v }
func (r *validationRecorder) SetMinLength(v int64)         { r.minLength = &v }
func (r *validationRecorder) SetMaxLength(v int64)         { r.maxLength = &v }
func (r *validationRecorder) SetPattern(v string)          { r.pattern = v }
func (r *validationRecorder) SetUnique(v bool)             { r.unique = &v }
func (r *validationRecorder) SetEnum(v string)             { r.enum = v }
func (r *validationRecorder) SetDefault(v any)             { r.defaultVal = v }
func (r *validationRecorder) SetExample(v any)             { r.exampleVal = v }
func (r *validationRecorder) SetCollectionFormat(v string) { r.collectionFormat = v }

func TestSetMaximum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		line      string
		wantMatch bool
		wantVal   float64
		exclusive bool
	}{
		{"inclusive", "maximum: 100", true, 100, false},
		{"exclusive", "maximum: < 100", true, 100, true},
		{"decimal", "maximum: 99.5", true, 99.5, false},
		{"negative", "maximum: -10", true, -10, false},
		{"no match", "something else", false, 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &validationRecorder{}
			sm := NewSetMaximum(rec)
			assert.EqualT(t, tc.wantMatch, sm.Matches(tc.line))
			if tc.wantMatch {
				require.NoError(t, sm.Parse([]string{tc.line}))
				require.NotNil(t, rec.maximum)
				assert.EqualT(t, tc.wantVal, *rec.maximum)
				assert.EqualT(t, tc.exclusive, rec.exclusiveMaximum)
			}
		})
	}

	t.Run("empty lines", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetMaximum(rec)
		require.NoError(t, sm.Parse(nil))
		require.NoError(t, sm.Parse([]string{}))
		require.NoError(t, sm.Parse([]string{""}))
		assert.Nil(t, rec.maximum)
	})

	t.Run("parse error", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetMaximum(rec)
		// Force a match with a non-numeric value via raw regex
		require.NoError(t, sm.Parse([]string{"maximum: not-a-number"}))
		assert.Nil(t, rec.maximum) // no match because regex won't capture non-numeric
	})
}

func TestSetMinimum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		line      string
		wantMatch bool
		wantVal   float64
		exclusive bool
	}{
		{"inclusive", "minimum: 0", true, 0, false},
		{"exclusive", "minimum: > 0", true, 0, true},
		{"decimal", "min: 1.5", true, 1.5, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &validationRecorder{}
			sm := NewSetMinimum(rec)
			assert.EqualT(t, tc.wantMatch, sm.Matches(tc.line))
			if tc.wantMatch {
				require.NoError(t, sm.Parse([]string{tc.line}))
				require.NotNil(t, rec.minimum)
				assert.EqualT(t, tc.wantVal, *rec.minimum)
				assert.EqualT(t, tc.exclusive, rec.exclusiveMinimum)
			}
		})
	}
}

func TestSetMultipleOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		line      string
		wantMatch bool
		wantVal   float64
	}{
		{"integer", "multiple of: 5", true, 5},
		{"decimal", "Multiple Of: 0.5", true, 0.5},
		{"no match", "something else", false, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &validationRecorder{}
			sm := NewSetMultipleOf(rec)
			assert.EqualT(t, tc.wantMatch, sm.Matches(tc.line))
			if tc.wantMatch {
				require.NoError(t, sm.Parse([]string{tc.line}))
				require.NotNil(t, rec.multipleOf)
				assert.EqualT(t, tc.wantVal, *rec.multipleOf)
			}
		})
	}
}

func TestSetMaxItems(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	sm := NewSetMaxItems(rec)
	assert.TrueT(t, sm.Matches("max items: 10"))
	require.NoError(t, sm.Parse([]string{"max items: 10"}))
	require.NotNil(t, rec.maxItems)
	assert.EqualT(t, int64(10), *rec.maxItems)
}

func TestSetMinItems(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	sm := NewSetMinItems(rec)
	assert.TrueT(t, sm.Matches("min items: 1"))
	require.NoError(t, sm.Parse([]string{"min items: 1"}))
	require.NotNil(t, rec.minItems)
	assert.EqualT(t, int64(1), *rec.minItems)
}

func TestSetMaxLength(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	sm := NewSetMaxLength(rec)
	assert.TrueT(t, sm.Matches("max length: 255"))
	require.NoError(t, sm.Parse([]string{"max length: 255"}))
	require.NotNil(t, rec.maxLength)
	assert.EqualT(t, int64(255), *rec.maxLength)
}

func TestSetMinLength(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	sm := NewSetMinLength(rec)
	assert.TrueT(t, sm.Matches("min length: 1"))
	require.NoError(t, sm.Parse([]string{"min length: 1"}))
	require.NotNil(t, rec.minLength)
	assert.EqualT(t, int64(1), *rec.minLength)
}

func TestSetPattern(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	sm := NewSetPattern(rec)
	assert.TrueT(t, sm.Matches("pattern: ^\\w+$"))
	require.NoError(t, sm.Parse([]string{"pattern: ^\\w+$"}))
	assert.EqualT(t, "^\\w+$", rec.pattern)
}

func TestSetCollectionFormat(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	sm := NewSetCollectionFormat(rec)
	assert.TrueT(t, sm.Matches("collection format: csv"))
	require.NoError(t, sm.Parse([]string{"collection format: csv"}))
	assert.EqualT(t, "csv", rec.collectionFormat)
}

func TestSetUnique(t *testing.T) {
	t.Parallel()

	tests := []struct {
		line string
		want bool
	}{
		{"unique: true", true},
		{"unique: false", false},
	}

	for _, tc := range tests {
		t.Run(tc.line, func(t *testing.T) {
			rec := &validationRecorder{}
			su := NewSetUnique(rec)
			assert.TrueT(t, su.Matches(tc.line))
			require.NoError(t, su.Parse([]string{tc.line}))
			require.NotNil(t, rec.unique)
			assert.EqualT(t, tc.want, *rec.unique)
		})
	}

	t.Run("parse error", func(t *testing.T) {
		rec := &validationRecorder{}
		su := NewSetUnique(rec)
		// unique: accepts only true/false so non-bool won't match
		assert.FalseT(t, su.Matches("unique: maybe"))
	})
}

func TestSetRequiredParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		line string
		want bool
	}{
		{"required: true", true},
		{"required: false", false},
	}

	for _, tc := range tests {
		t.Run(tc.line, func(t *testing.T) {
			param := new(oaispec.Parameter)
			su := NewSetRequiredParam(param)
			assert.TrueT(t, su.Matches(tc.line))
			require.NoError(t, su.Parse([]string{tc.line}))
			assert.EqualT(t, tc.want, param.Required)
		})
	}

	t.Run("empty", func(t *testing.T) {
		param := new(oaispec.Parameter)
		su := NewSetRequiredParam(param)
		require.NoError(t, su.Parse(nil))
		assert.FalseT(t, param.Required)
	})
}

func TestSetReadOnlySchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		line string
		want bool
	}{
		{"read only: true", true},
		{"readOnly: true", true},
		{"read-only: false", false},
	}

	for _, tc := range tests {
		t.Run(tc.line, func(t *testing.T) {
			schema := new(oaispec.Schema)
			su := NewSetReadOnlySchema(schema)
			assert.TrueT(t, su.Matches(tc.line))
			require.NoError(t, su.Parse([]string{tc.line}))
			assert.EqualT(t, tc.want, schema.ReadOnly)
		})
	}
}

func TestSetRequiredSchema(t *testing.T) {
	t.Parallel()

	t.Run("set required true", func(t *testing.T) {
		schema := new(oaispec.Schema)
		su := NewSetRequiredSchema(schema, "name")
		require.NoError(t, su.Parse([]string{"required: true"}))
		assert.Equal(t, []string{"name"}, schema.Required)
	})

	t.Run("set required false removes", func(t *testing.T) {
		schema := &oaispec.Schema{}
		schema.Required = []string{"name", "age"}
		su := NewSetRequiredSchema(schema, "name")
		require.NoError(t, su.Parse([]string{"required: false"}))
		assert.Equal(t, []string{"age"}, schema.Required)
	})

	t.Run("set required true idempotent", func(t *testing.T) {
		schema := &oaispec.Schema{}
		schema.Required = []string{"name"}
		su := NewSetRequiredSchema(schema, "name")
		require.NoError(t, su.Parse([]string{"required: true"}))
		assert.Equal(t, []string{"name"}, schema.Required)
	})

	t.Run("set required false not present", func(t *testing.T) {
		schema := new(oaispec.Schema)
		su := NewSetRequiredSchema(schema, "name")
		require.NoError(t, su.Parse([]string{"required: false"}))
		assert.Empty(t, schema.Required)
	})

	t.Run("empty lines", func(t *testing.T) {
		schema := new(oaispec.Schema)
		su := NewSetRequiredSchema(schema, "name")
		require.NoError(t, su.Parse(nil))
		require.NoError(t, su.Parse([]string{""}))
	})

	t.Run("no match in line", func(t *testing.T) {
		schema := new(oaispec.Schema)
		su := NewSetRequiredSchema(schema, "name")
		require.NoError(t, su.Parse([]string{"something else"}))
		assert.Empty(t, schema.Required)
	})
}

func TestSetDefault(t *testing.T) {
	t.Parallel()

	t.Run("string type", func(t *testing.T) {
		rec := &validationRecorder{}
		scheme := &oaispec.SimpleSchema{Type: "string"}
		sd := NewSetDefault(scheme, rec)
		assert.TrueT(t, sd.Matches("default: hello"))
		require.NoError(t, sd.Parse([]string{"default: hello"}))
		assert.EqualT(t, "hello", rec.defaultVal)
	})

	t.Run("integer type", func(t *testing.T) {
		rec := &validationRecorder{}
		scheme := &oaispec.SimpleSchema{Type: "integer"}
		sd := NewSetDefault(scheme, rec)
		require.NoError(t, sd.Parse([]string{"default: 42"}))
		assert.EqualT(t, 42, rec.defaultVal)
	})

	t.Run("empty", func(t *testing.T) {
		rec := &validationRecorder{}
		scheme := &oaispec.SimpleSchema{Type: "string"}
		sd := NewSetDefault(scheme, rec)
		require.NoError(t, sd.Parse(nil))
		assert.Nil(t, rec.defaultVal)
	})
}

func TestSetExample(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	scheme := &oaispec.SimpleSchema{Type: "string"}
	se := NewSetExample(scheme, rec)
	assert.TrueT(t, se.Matches("example: foobar"))
	require.NoError(t, se.Parse([]string{"example: foobar"}))
	assert.EqualT(t, "foobar", rec.exampleVal)
}

func TestSetDiscriminator(t *testing.T) {
	t.Parallel()

	t.Run("set true", func(t *testing.T) {
		schema := new(oaispec.Schema)
		sd := NewSetDiscriminator(schema, "kind")
		assert.TrueT(t, sd.Matches("discriminator: true"))
		require.NoError(t, sd.Parse([]string{"discriminator: true"}))
		assert.EqualT(t, "kind", schema.Discriminator)
	})

	t.Run("set false clears", func(t *testing.T) {
		schema := &oaispec.Schema{}
		schema.Discriminator = "kind"
		sd := NewSetDiscriminator(schema, "kind")
		require.NoError(t, sd.Parse([]string{"discriminator: false"}))
		assert.EqualT(t, "", schema.Discriminator)
	})

	t.Run("set false different field", func(t *testing.T) {
		schema := &oaispec.Schema{}
		schema.Discriminator = "type"
		sd := NewSetDiscriminator(schema, "kind")
		require.NoError(t, sd.Parse([]string{"discriminator: false"}))
		assert.EqualT(t, "type", schema.Discriminator) // unchanged
	})
}

func TestWithItemsPrefixLevel(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	sm := NewSetMaximum(rec, WithItemsPrefixLevel(0))
	line := "items.maximum: 100"
	assert.TrueT(t, sm.Matches(line))
	require.NoError(t, sm.Parse([]string{line}))
	require.NotNil(t, rec.maximum)
	assert.EqualT(t, float64(100), *rec.maximum)

	// Level 1 requires "items.items."
	rec2 := &validationRecorder{}
	sm2 := NewSetMinimum(rec2, WithItemsPrefixLevel(1))
	line2 := "items.items.minimum: 5"
	assert.TrueT(t, sm2.Matches(line2))
	require.NoError(t, sm2.Parse([]string{line2}))
	require.NotNil(t, rec2.minimum)
	assert.EqualT(t, float64(5), *rec2.minimum)
}

func TestSetEnum(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	se := NewSetEnum(rec)
	line := "enum: " + `["a","b","c"]`
	assert.TrueT(t, se.Matches(line))
	require.NoError(t, se.Parse([]string{line}))
	assert.EqualT(t, `["a","b","c"]`, rec.enum)

	t.Run("empty", func(t *testing.T) {
		rec := &validationRecorder{}
		se := NewSetEnum(rec)
		require.NoError(t, se.Parse(nil))
		require.NoError(t, se.Parse([]string{""}))
		assert.EqualT(t, "", rec.enum)
	})
}

// TestPrefixRxOption_AllConstructors covers the WithItemsPrefixLevel loop body
// in every validation constructor that accepts PrefixRxOption.
func TestPrefixRxOption_AllConstructors(t *testing.T) {
	t.Parallel()

	prefix := WithItemsPrefixLevel(0)

	t.Run("SetMultipleOf", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetMultipleOf(rec, prefix)
		line := "items.multiple of: 3"
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		require.NotNil(t, rec.multipleOf)
		assert.EqualT(t, float64(3), *rec.multipleOf)
	})

	t.Run("SetMaxItems", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetMaxItems(rec, prefix)
		line := "items.max items: 10"
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		require.NotNil(t, rec.maxItems)
		assert.EqualT(t, int64(10), *rec.maxItems)
	})

	t.Run("SetMinItems", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetMinItems(rec, prefix)
		line := "items.min items: 1"
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		require.NotNil(t, rec.minItems)
		assert.EqualT(t, int64(1), *rec.minItems)
	})

	t.Run("SetMaxLength", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetMaxLength(rec, prefix)
		line := "items.max length: 100"
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		require.NotNil(t, rec.maxLength)
		assert.EqualT(t, int64(100), *rec.maxLength)
	})

	t.Run("SetMinLength", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetMinLength(rec, prefix)
		line := "items.min length: 1"
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		require.NotNil(t, rec.minLength)
		assert.EqualT(t, int64(1), *rec.minLength)
	})

	t.Run("SetPattern", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetPattern(rec, prefix)
		line := "items.pattern: ^[a-z]+$"
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		assert.EqualT(t, "^[a-z]+$", rec.pattern)
	})

	t.Run("SetCollectionFormat", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetCollectionFormat(rec, prefix)
		line := "items.collection format: pipes"
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		assert.EqualT(t, "pipes", rec.collectionFormat)
	})

	t.Run("SetUnique", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetUnique(rec, prefix)
		line := "items.unique: true"
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		require.NotNil(t, rec.unique)
		assert.TrueT(t, *rec.unique)
	})

	t.Run("SetDefault", func(t *testing.T) {
		rec := &validationRecorder{}
		scheme := &oaispec.SimpleSchema{Type: "string"}
		sm := NewSetDefault(scheme, rec, prefix)
		line := "items.default: hello"
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		assert.EqualT(t, "hello", rec.defaultVal)
	})

	t.Run("SetExample", func(t *testing.T) {
		rec := &validationRecorder{}
		scheme := &oaispec.SimpleSchema{Type: "string"}
		sm := NewSetExample(scheme, rec, prefix)
		line := "items.example: world"
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		assert.EqualT(t, "world", rec.exampleVal)
	})

	t.Run("SetEnum", func(t *testing.T) {
		rec := &validationRecorder{}
		sm := NewSetEnum(rec, prefix)
		line := `items.enum: ["x","y"]`
		assert.TrueT(t, sm.Matches(line))
		require.NoError(t, sm.Parse([]string{line}))
		assert.EqualT(t, `["x","y"]`, rec.enum)
	})
}

func TestSetDefault_ParseError(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	scheme := &oaispec.SimpleSchema{Type: "integer"}
	sd := NewSetDefault(scheme, rec)
	err := sd.Parse([]string{"default: not-a-number"})
	require.Error(t, err)
	assert.Nil(t, rec.defaultVal)
}

func TestSetExample_ParseError(t *testing.T) {
	t.Parallel()

	rec := &validationRecorder{}
	scheme := &oaispec.SimpleSchema{Type: "integer"}
	se := NewSetExample(scheme, rec)
	err := se.Parse([]string{"example: not-a-number"})
	require.Error(t, err)
	assert.Nil(t, rec.exampleVal)
}

func TestSetRequiredSchema_Matches(t *testing.T) {
	t.Parallel()

	su := NewSetRequiredSchema(new(oaispec.Schema), "name")
	assert.TrueT(t, su.Matches("required: true"))
	assert.TrueT(t, su.Matches("Required: false"))
	assert.FalseT(t, su.Matches("something else"))
}

// strictMockValidationBuilder returns a MockValidationBuilder whose Set* methods
// fail the test if called. Use this in tests that assert no mutation happened
// (empty-input tolerance, overflow errors, etc.).
func strictMockValidationBuilder(t *testing.T) *mocks.MockValidationBuilder {
	t.Helper()
	fail := func(name string) func(...any) {
		return func(args ...any) { t.Fatalf("%s should not be called (args: %v)", name, args) }
	}
	m := &mocks.MockValidationBuilder{}
	m.SetMaximumFunc = func(v float64, exclusive bool) { fail("SetMaximum")(v, exclusive) }
	m.SetMinimumFunc = func(v float64, exclusive bool) { fail("SetMinimum")(v, exclusive) }
	m.SetMultipleOfFunc = func(v float64) { fail("SetMultipleOf")(v) }
	m.SetMaxItemsFunc = func(v int64) { fail("SetMaxItems")(v) }
	m.SetMinItemsFunc = func(v int64) { fail("SetMinItems")(v) }
	m.SetMaxLengthFunc = func(v int64) { fail("SetMaxLength")(v) }
	m.SetMinLengthFunc = func(v int64) { fail("SetMinLength")(v) }
	m.SetPatternFunc = func(v string) { fail("SetPattern")(v) }
	m.SetUniqueFunc = func(v bool) { fail("SetUnique")(v) }
	m.SetEnumFunc = func(v string) { fail("SetEnum")(v) }
	m.SetDefaultFunc = func(v any) { fail("SetDefault")(v) }
	m.SetExampleFunc = func(v any) { fail("SetExample")(v) }
	return m
}

// TestValidationParsers_EmptyInputTolerance pins the defensive-guard
// contract documented in the D.5 post-mortem: every Parse(lines) tolerates
// nil / empty-slice / single-empty-string input without panic and without
// mutating its target. Uses MockValidationBuilder (Set* funcs fail on call)
// to prove no side effect.
func TestValidationParsers_EmptyInputTolerance(t *testing.T) {
	t.Parallel()

	emptyInputs := [][]string{nil, {}, {""}}

	cases := []struct {
		name    string
		factory func(*testing.T) ifaces.ValueParser
	}{
		{"SetMaximum", func(t *testing.T) ifaces.ValueParser { return NewSetMaximum(strictMockValidationBuilder(t)) }},
		{"SetMinimum", func(t *testing.T) ifaces.ValueParser { return NewSetMinimum(strictMockValidationBuilder(t)) }},
		{"SetMultipleOf", func(t *testing.T) ifaces.ValueParser { return NewSetMultipleOf(strictMockValidationBuilder(t)) }},
		{"SetMaxItems", func(t *testing.T) ifaces.ValueParser { return NewSetMaxItems(strictMockValidationBuilder(t)) }},
		{"SetMinItems", func(t *testing.T) ifaces.ValueParser { return NewSetMinItems(strictMockValidationBuilder(t)) }},
		{"SetMaxLength", func(t *testing.T) ifaces.ValueParser { return NewSetMaxLength(strictMockValidationBuilder(t)) }},
		{"SetMinLength", func(t *testing.T) ifaces.ValueParser { return NewSetMinLength(strictMockValidationBuilder(t)) }},
		{"SetPattern", func(t *testing.T) ifaces.ValueParser { return NewSetPattern(strictMockValidationBuilder(t)) }},
		{"SetUnique", func(t *testing.T) ifaces.ValueParser { return NewSetUnique(strictMockValidationBuilder(t)) }},
		{"SetExample", func(t *testing.T) ifaces.ValueParser {
			scheme := &oaispec.SimpleSchema{Type: "string"}
			return NewSetExample(scheme, strictMockValidationBuilder(t))
		}},
		{"SetCollectionFormat", func(t *testing.T) ifaces.ValueParser {
			// OperationValidationBuilder — use the op-variant mock, fail-all.
			m := &mocks.MockOperationValidationBuilder{
				SetCollectionFormatFunc: func(v string) { t.Fatalf("SetCollectionFormat should not be called (arg: %s)", v) },
			}
			return NewSetCollectionFormat(m)
		}},
		{"SetReadOnlySchema", func(_ *testing.T) ifaces.ValueParser { return NewSetReadOnlySchema(new(oaispec.Schema)) }},
		{"SetDiscriminator", func(_ *testing.T) ifaces.ValueParser { return NewSetDiscriminator(new(oaispec.Schema), "kind") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.factory(t)
			for _, in := range emptyInputs {
				require.NoError(t, p.Parse(in))
			}
		})
	}
}

// TestValidationParsers_NumericOverflow pins the overflow defence we kept
// in D.5: the regex captures \p{N}+ (any-length digit string), which matches
// values beyond int64 / float64 range. strconv.ParseInt / ParseFloat returns
// ErrRange in those cases, and the parser must propagate the error without
// invoking the target builder. See .claude/plans/dead-code-cleanup.md D.5
// post-mortem for the rationale.
func TestValidationParsers_NumericOverflow(t *testing.T) {
	t.Parallel()

	// int64 max is 9223372036854775807 (19 digits); 20+ 9's overflows.
	intOverflow := strings.Repeat("9", 25)
	// float64 max is ~1.8e308 in magnitude; 400 9's in decimal notation
	// overflows ParseFloat (returns +Inf, ErrRange).
	floatOverflow := strings.Repeat("9", 400)

	cases := []struct {
		name string
		line string
		newP func(*testing.T) ifaces.ValueParser
	}{
		{
			name: "SetMaximum float overflow",
			line: "maximum: " + floatOverflow,
			newP: func(t *testing.T) ifaces.ValueParser { return NewSetMaximum(strictMockValidationBuilder(t)) },
		},
		{
			name: "SetMinimum float overflow",
			line: "minimum: " + floatOverflow,
			newP: func(t *testing.T) ifaces.ValueParser { return NewSetMinimum(strictMockValidationBuilder(t)) },
		},
		{
			name: "SetMultipleOf float overflow",
			line: "multiple of: " + floatOverflow,
			newP: func(t *testing.T) ifaces.ValueParser { return NewSetMultipleOf(strictMockValidationBuilder(t)) },
		},
		{
			name: "SetMaxItems int overflow",
			line: "max items: " + intOverflow,
			newP: func(t *testing.T) ifaces.ValueParser { return NewSetMaxItems(strictMockValidationBuilder(t)) },
		},
		{
			name: "SetMinItems int overflow",
			line: "min items: " + intOverflow,
			newP: func(t *testing.T) ifaces.ValueParser { return NewSetMinItems(strictMockValidationBuilder(t)) },
		},
		{
			name: "SetMaxLength int overflow",
			line: "max length: " + intOverflow,
			newP: func(t *testing.T) ifaces.ValueParser { return NewSetMaxLength(strictMockValidationBuilder(t)) },
		},
		{
			name: "SetMinLength int overflow",
			line: "min length: " + intOverflow,
			newP: func(t *testing.T) ifaces.ValueParser { return NewSetMinLength(strictMockValidationBuilder(t)) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.newP(t)
			require.TrueT(t, p.Matches(tc.line), "regex must match overflow input; otherwise the guard we're testing is dead")
			err := p.Parse([]string{tc.line})
			require.Error(t, err, "expected ParseInt/ParseFloat ErrRange")
		})
	}
}
