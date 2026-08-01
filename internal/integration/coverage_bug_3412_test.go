// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug3412 covers go-swagger issue #3412 ("negative enum values are dropped"):
// `PanLeft PanDirection = -1` is a unary expression in the Go grammar, not a literal, so the
// scanner's BasicLit-only value extraction skipped it and the emitted enum was [0, 1].
//
// The enum type is consumed from all three enum-carrying targets — response header, schema
// property and non-body parameter — so the negative value must survive each target's coercion,
// not merely the scan. A float enum covers the FLOAT branch, and a non-decimal enum covers the
// base-detected integer forms.
func TestCoverage_Bug3412(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3412/negative/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	signedInts := []any{int64(-1), int64(0), int64(1)}
	// TiltFlat is written `= 0`, an INT literal, but a constant's kind follows its declared type —
	// float64 here — not the shape of the literal, so the member is 0.0 and the enum stays a
	// number enum.
	signedFloats := []any{-0.5, float64(0), 0.5}

	t.Run("schema property keeps the negative member", func(t *testing.T) {
		props := doc.Definitions["ControlState"].Properties

		assert.Equal(t, signedInts, props["pan"].Enum)
		assert.Contains(t, props["pan"].Extensions["x-go-enum-desc"], "-1 PanLeft")

		assert.Equal(t, signedFloats, props["tilt"].Enum)
		assert.Contains(t, props["tilt"].Extensions["x-go-enum-desc"], "-0.5 TiltDown")
	})

	t.Run("non-body parameter keeps the negative member", func(t *testing.T) {
		op := doc.Paths.Paths["/pan/{pan}"].Get
		require.NotNil(t, op, "GET /pan/{pan} operation must be present")
		require.Len(t, op.Parameters, 1)
		param := op.Parameters[0]
		require.Equal(t, "query", param.In)

		assert.Equal(t, signedInts, param.Enum)
		assert.Contains(t, param.Description, "-1 PanLeft")
	})

	t.Run("response header keeps the negative member", func(t *testing.T) {
		headers := doc.Responses["ControlParams"].Headers

		assert.Equal(t, signedInts, headers["pan"].Enum)
		assert.Equal(t, signedFloats, headers["tilt"].Enum)
	})

	t.Run("non-decimal integer literals resolve to their value", func(t *testing.T) {
		mask := doc.Definitions["ControlState"].Properties["mask"]

		assert.Equal(t, []any{int64(42), int64(42), int64(42), int64(1000)}, mask.Enum,
			"hex, binary and octal forms resolve like the decimal one; digit separators are ignored")
	})

	t.Run("an unsigned member above MaxInt64 survives the signed parse", func(t *testing.T) {
		aperture := doc.Definitions["ControlState"].Properties["aperture"]

		assert.Equal(t, []any{int64(0), uint64(18446744073709551615)}, aperture.Enum)
	})

	// The enum's type and format come from the DECLARED Go type. They used to be read off the first
	// parsed value, which collapsed every width to int64/double and — worse — let the declaration
	// order of the const block decide the type of the whole enum.
	t.Run("type and format follow the declared Go type", func(t *testing.T) {
		props := doc.Definitions["ControlState"].Properties

		for _, tc := range []struct {
			property string
			goType   string
			typ      string
			format   string
		}{
			{property: "pan", goType: "int8", typ: "integer", format: "int8"},
			{property: "tilt", goType: "float64", typ: "number", format: "double"},
			{property: "mask", goType: "int64", typ: "integer", format: "int64"},
			{property: "zoom", goType: "float32", typ: "number", format: "float"},
			{property: "aperture", goType: "uint64", typ: "integer", format: "uint64"},
		} {
			t.Run(tc.goType, func(t *testing.T) {
				assert.Equal(t, []string{tc.typ}, []string(props[tc.property].Type))
				assert.Equal(t, tc.format, props[tc.property].Format)
			})
		}
	})

	t.Run("an integer literal in a float enum does not make the enum an integer one", func(t *testing.T) {
		zoom := doc.Definitions["ControlState"].Properties["zoom"]

		// ZoomNone is `= 0`, written FIRST in the const block.
		assert.Equal(t, []string{"number"}, []string(zoom.Type))
		assert.Equal(t, []any{float64(0), -1.5, 1.5}, zoom.Enum)
	})

	t.Run("the declared type reaches the SimpleSchema targets too", func(t *testing.T) {
		header := doc.Responses["ControlParams"].Headers["pan"]
		assert.Equal(t, "integer", header.Type)
		assert.Equal(t, "int8", header.Format)

		param := doc.Paths.Paths["/pan/{pan}"].Get.Parameters[0]
		assert.Equal(t, "integer", param.Type)
		assert.Equal(t, "int8", param.Format)
	})

	// The subtests above read the properties the fix is about; the golden pins everything else the
	// fixture emits alongside them, so a change nobody asserted on still has to be looked at.
	scantest.CompareOrDumpJSON(t, doc, "bugs_3412_negative.json")
}

// TestCoverage_Bug3412_ConstForms covers the const shapes that carry no readable value in their own
// syntax, and which a literal-syntax reader therefore cannot see at all: `iota` (where the implicit
// specs have neither a type nor a value), constant expressions, references to earlier members, rune
// literals, `true`/`false` (identifiers, not literals — Go has no boolean literal token), and the
// raw / escaped string forms.
//
// Values come from the type-checker, which evaluated all of them exactly; the schema type still
// comes from the declared Go type.
func TestCoverage_Bug3412_ConstForms(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3412/constforms/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	props := doc.Definitions["Settings"].Properties

	for _, tc := range []struct {
		name     string
		property string
		typ      string
		format   string
		enum     []any
	}{
		{
			name:     "iota block: the implicit specs carry neither type nor value",
			property: "weekday",
			typ:      "integer", format: "int64",
			enum: []any{int64(0), int64(1), int64(2)},
		},
		{
			name:     "constant expression and reference to an earlier member",
			property: "level",
			typ:      "integer", format: "int64",
			enum: []any{int64(8), int64(16)},
		},
		{
			name:     "rune literals are integer constants",
			property: "letter",
			typ:      "integer", format: "int32",
			enum: []any{int64('a'), int64('\t')},
		},
		{
			name:     "true and false are identifiers, not literals",
			property: "toggle",
			typ:      "boolean", format: "",
			enum: []any{true, false},
		},
		{
			name:     "raw and escaped strings lose their delimiters and resolve escapes",
			property: "label",
			typ:      "string", format: "",
			enum: []any{"raw", "a\tb"},
		},
		{
			name:     "a byte enum mixing an integer and a rune literal",
			property: "byte",
			typ:      "integer", format: "uint8",
			enum: []any{int64(0), int64('x')},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, []string{tc.typ}, []string(props[tc.property].Type))
			assert.Equal(t, tc.format, props[tc.property].Format)
			assert.Equal(t, tc.enum, props[tc.property].Enum)
		})
	}

	t.Run("a const of a same-named imported type is not a member", func(t *testing.T) {
		// `const ForeignDay foreign.Weekday = 13` sits in the annotated package but belongs to another
		// package's Weekday. Membership is type identity, not name.
		assert.NotContains(t, props["weekday"].Enum, int64(13))
	})

	t.Run("the per-value name mapping follows the evaluated value", func(t *testing.T) {
		assert.Equal(t, "0 Sunday\n1 Monday\n2 Tuesday",
			props["weekday"].Extensions["x-go-enum-desc"])
		assert.Equal(t, "97 LetterA\n9 LetterTab",
			props["letter"].Extensions["x-go-enum-desc"])
	})

	scantest.CompareOrDumpJSON(t, doc, "bugs_3412_constforms.json")
}

// TestCoverage_Bug3412_SimpleSchema pins the whole propagation surface of a `swagger:enum` type
// with a whole-spec golden.
//
// The enum reaches the spec through several builder paths: a schema property and its array items,
// but also the SimpleSchema targets — `in: path` / `query` / `header` / `formData`, the `items` of
// an array-typed parameter, and response headers with their own `items` — where OAS v2 forbids the
// `$ref` a definition would use, so the members and the declared type/format must be written
// inline. Each is a distinct path, and a per-property assertion in one of them says nothing about
// the others; the golden is what makes a regression in any single target visible.
func TestCoverage_Bug3412_SimpleSchema(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3412/simpleschema/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "bugs_3412_simpleschema.json")
}

// TestCoverage_Bug3412_StrfmtEnum covers an enum whose declared type is written over another NAMED
// type. `type Kind strfmt.UUID` has `string` as its go/types underlying, so the `uuid` format the
// strfmt declaration carries is absent from the view the enum arm types from — while the very same
// type without the annotation keeps it, since the ordinary path resolves the declaration's
// right-hand side instead. Annotating a type as an enum must not cost the author their format.
func TestCoverage_Bug3412_StrfmtEnum(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3412/strfmtenum/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	props := doc.Definitions["Labels"].Properties

	t.Run("the format of the type the enum is written over survives", func(t *testing.T) {
		assert.Equal(t, []string{"string"}, []string(props["kind"].Type))
		assert.Equal(t, "uuid", props["kind"].Format)
		assert.Equal(t, []any{
			"0a8bcf1e-0000-0000-0000-000000000000",
			"0a8bcf1e-1111-1111-1111-111111111111",
		}, props["kind"].Enum)
	})

	t.Run("an indirection chain resolves like a single redefinition", func(t *testing.T) {
		assert.Equal(t, []string{"string"}, []string(props["contact"].Type))
		assert.Equal(t, "email", props["contact"].Format)
		assert.Equal(t, []any{"support@example.com"}, props["contact"].Enum)
	})

	t.Run("an enum straight over a basic type keeps the basic format", func(t *testing.T) {
		assert.Equal(t, []string{"string"}, []string(props["plain"].Type))
		assert.Empty(t, props["plain"].Format)
		assert.Equal(t, []any{"on"}, props["plain"].Enum)
	})

	scantest.CompareOrDumpJSON(t, doc, "bugs_3412_strfmtenum.json")
}
