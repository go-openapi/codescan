// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package validations_test

import (
	"testing"

	"github.com/go-openapi/codescan/internal/builders/validations"
	"github.com/go-openapi/testify/v2/assert"
)

func TestIsFormatCompatible(t *testing.T) {
	tests := []struct {
		name       string
		schemaType string
		format     string
		want       bool
	}{
		// string accepts any format (strfmt is string-oriented).
		{"string + uuid", "string", "uuid", true},
		{"string + date-time", "string", "date-time", true},
		{"string + int64", "string", "int64", true},
		{"string + email", "string", "email", true},

		// empty format is trivially compatible (nothing to apply).
		{"integer + empty", "integer", "", true},
		{"boolean + empty", "boolean", "", true},

		// integer accepts int{n} / uint{n}, rejects floats and string formats.
		{"integer + int32", "integer", "int32", true},
		{"integer + int64", "integer", "int64", true},
		{"integer + int8", "integer", "int8", true},
		{"integer + uint32", "integer", "uint32", true},
		{"integer + uint64", "integer", "uint64", true},
		{"integer + float64", "integer", "float64", false},
		{"integer + double", "integer", "double", false},
		{"integer + uuid", "integer", "uuid", false},

		// number accepts int{n} / uint{n} AND float widths.
		{"number + int64", "number", "int64", true},
		{"number + uint32", "number", "uint32", true},
		{"number + float", "number", "float", true},
		{"number + double", "number", "double", true},
		{"number + float32", "number", "float32", true},
		{"number + float64", "number", "float64", true},
		{"number + uuid", "number", "uuid", false},

		// other types take no format.
		{"boolean + int64", "boolean", "int64", false},
		{"object + anything", "object", "x", false},
		{"array + int64", "array", "int64", false},
		{"file + binary", "file", "binary", false},

		// unknown type is accepted best-effort (no type to check).
		{"empty type + int64", "", "int64", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, hint := validations.IsFormatCompatible(tc.schemaType, tc.format)
			assert.Equal(t, tc.want, ok)
			if tc.want {
				assert.Empty(t, hint, "compatible result must not carry a hint")
			} else {
				assert.NotEmpty(t, hint, "incompatible result must carry a hint")
				assert.Contains(t, hint, tc.format)
				assert.Contains(t, hint, tc.schemaType)
			}
		})
	}
}
