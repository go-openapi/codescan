// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package validations_test

import (
	"testing"

	"github.com/go-openapi/codescan/internal/builders/validations"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
)

func TestIsLegalForType_PatternOnlyOnString(t *testing.T) {
	kw := grammar.Keyword{Name: "pattern"}

	ok, _ := validations.IsLegalForType(kw, "string")
	assert.True(t, ok, "pattern legal on string")

	ok, hint := validations.IsLegalForType(kw, "integer")
	assert.False(t, ok)
	assert.Contains(t, hint, "string")
	assert.Contains(t, hint, "integer")
}

func TestIsLegalForType_NumericOnNumberOrInteger(t *testing.T) {
	for _, kwName := range []string{"maximum", "minimum", "multipleOf"} {
		kw := grammar.Keyword{Name: kwName}

		okNum, _ := validations.IsLegalForType(kw, "number")
		assert.True(t, okNum, "%s legal on number", kwName)

		okInt, _ := validations.IsLegalForType(kw, "integer")
		assert.True(t, okInt, "%s legal on integer", kwName)

		okStr, _ := validations.IsLegalForType(kw, "string")
		assert.False(t, okStr, "%s NOT legal on string", kwName)
	}
}

func TestIsLegalForType_StringLengthOnlyOnString(t *testing.T) {
	for _, kwName := range []string{"minLength", "maxLength"} {
		kw := grammar.Keyword{Name: kwName}

		ok, _ := validations.IsLegalForType(kw, "string")
		assert.True(t, ok, "%s legal on string", kwName)

		ok, _ = validations.IsLegalForType(kw, "integer")
		assert.False(t, ok, "%s NOT legal on integer", kwName)
	}
}

func TestIsLegalForType_ArrayConstraintsOnlyOnArray(t *testing.T) {
	for _, kwName := range []string{"minItems", "maxItems", "uniqueItems"} {
		kw := grammar.Keyword{Name: kwName}

		ok, _ := validations.IsLegalForType(kw, "array")
		assert.True(t, ok, "%s legal on array", kwName)

		ok, _ = validations.IsLegalForType(kw, "string")
		assert.False(t, ok, "%s NOT legal on string", kwName)

		ok, _ = validations.IsLegalForType(kw, "object")
		assert.False(t, ok, "%s NOT legal on object", kwName)
	}
}

func TestIsLegalForType_ObjectConstraintsOnlyOnObject(t *testing.T) {
	for _, kwName := range []string{"minProperties", "maxProperties"} {
		kw := grammar.Keyword{Name: kwName}

		ok, _ := validations.IsLegalForType(kw, "object")
		assert.True(t, ok, "%s legal on object", kwName)

		ok, _ = validations.IsLegalForType(kw, "array")
		assert.False(t, ok, "%s NOT legal on array", kwName)
	}
}

func TestIsLegalForType_TypelessSchemaIsLenient(t *testing.T) {
	// Empty schemaType ("type unknown") is accepted — the caller decides whether to apply.
	kw := grammar.Keyword{Name: "pattern"}
	ok, hint := validations.IsLegalForType(kw, "")
	assert.True(t, ok)
	assert.Empty(t, hint)
}

func TestIsLegalForType_KeywordsWithoutRulesAlwaysLegal(t *testing.T) {
	// required / readOnly / deprecated / discriminator have no type constraint — they apply to any
	// type.
	for _, kwName := range []string{"required", "readOnly", "deprecated", "discriminator"} {
		kw := grammar.Keyword{Name: kwName}
		for _, schemaType := range []string{"string", "integer", "number", "object", "array"} {
			ok, _ := validations.IsLegalForType(kw, schemaType)
			assert.True(t, ok, "%s should be legal on %s", kwName, schemaType)
		}
	}
}

func TestIsLegalForType_DefaultExampleEnumLegalOnAnyType(t *testing.T) {
	// default / example / enum coerce against the schema's type+format via CoerceValue / CoerceEnum,
	// so they're always legal — type validation lives there, not here.
	for _, kwName := range []string{"default", "example", "enum"} {
		kw := grammar.Keyword{Name: kwName}
		for _, schemaType := range []string{"string", "integer", "number", "object", "array"} {
			ok, _ := validations.IsLegalForType(kw, schemaType)
			assert.True(t, ok, "%s should be legal on %s", kwName, schemaType)
		}
	}
}
