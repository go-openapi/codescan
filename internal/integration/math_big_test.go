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

// The math/big numbers are recognized by identity, so they never reach structural drilling.
//
// Each takes the shape encoding/json actually produces and accepts, and the three do not agree:
// big.Int has a MarshalJSON emitting a bare numeric literal — and json.Marshaler beats the
// MarshalText the same type also carries — so it is an `integer`, while big.Float and big.Rat have
// only MarshalText and therefore travel quoted ("3.5", "5/3"), so they are `string`.
func TestMathBig(t *testing.T) {
	doc, err := runScan(&codescan.Options{
		Packages:   []string{"./enhancements/math-big/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	t.Run("pointer and value render alike", func(t *testing.T) {
		// The core claim. encoding/json takes the address of an addressable field to reach a
		// pointer-receiver marshaller, so `big.Int` and `*big.Int` fields of a struct marshalled
		// through a pointer emit the same thing. They used to disagree: through a pointer all three
		// satisfied TextMarshaler and collapsed onto `string`, and by value none of them did, so they
		// were drilled into object definitions carrying math/big's godoc.
		props := doc.Definitions["BigModel"].Properties
		require.NotEmpty(t, props)

		for ptr, val := range map[string]string{
			"total":     "count",
			"magnitude": "distance",
			"share":     "proportion",
		} {
			assert.Equal(t, props[ptr].Type, props[val].Type,
				"%s (pointer) and %s (value) are the same Go type and must render alike", ptr, val)
			assert.Equal(t, props[ptr].Extensions["x-go-type"], props[val].Extensions["x-go-type"])
		}
	})

	t.Run("each type takes its own wire shape", func(t *testing.T) {
		props := doc.Definitions["BigModel"].Properties

		for field, want := range map[string]string{
			"total":      "integer", // *big.Int  — MarshalJSON wins: a bare number
			"count":      "integer", // big.Int
			"magnitude":  "string",  // *big.Float — MarshalText only: "3.5"
			"distance":   "string",  // big.Float
			"share":      "string",  // *big.Rat   — MarshalText only: "5/3"
			"proportion": "string",  // big.Rat
		} {
			require.Len(t, props[field].Type, 1, "%s must carry exactly one type", field)
			assert.Equal(t, want, props[field].Type[0], "%s has the wrong wire shape", field)
			assert.Empty(t, props[field].Format, "%s must claim no format: the precision is unbounded", field)
		}
	})

	t.Run("x-go-type records which number it was", func(t *testing.T) {
		// Neither answer can carry it: `integer` cannot say the value is unbounded rather than an
		// int64, and `string` cannot tell a decimal float from a quotient.
		props := doc.Definitions["BigModel"].Properties

		for field, want := range map[string]string{
			"total":     "math/big.Int",
			"count":     "math/big.Int",
			"magnitude": "math/big.Float",
			"share":     "math/big.Rat",
		} {
			assert.Equal(t, want, props[field].Extensions["x-go-type"], "%s must record its Go type", field)
		}
	})

	t.Run("math/big is never published as a definition", func(t *testing.T) {
		// The leak the recognizer closes: drilling a big.Int value field used to emit `Int`, `Float`
		// and `Rat` as object definitions carrying math/big's own godoc.
		for _, leaked := range []string{"Int", "Float", "Rat"} {
			assert.NotContains(t, doc.Definitions, leaked,
				"math/big.%s must not be published as a definition", leaked)
		}
	})

	t.Run("reached through a container", func(t *testing.T) {
		props := doc.Definitions["BigModel"].Properties

		require.NotNil(t, props["ledger"].Items)
		assert.Equal(t, []string{"integer"}, []string(props["ledger"].Items.Schema.Type))
		require.NotNil(t, props["weights"].AdditionalProperties)
		assert.Equal(t, []string{"string"}, []string(props["weights"].AdditionalProperties.Schema.Type))
	})

	t.Run("an explicit annotation still wins", func(t *testing.T) {
		props := doc.Definitions["OverriddenModel"].Properties

		// A format override adjusts the format; the recognizer's stamp survives.
		assert.Equal(t, "bigdecimal", props["principal"].Format)
		assert.Equal(t, "math/big.Float", props["principal"].Extensions["x-go-type"])

		// A type override replaces the schema outright; the stamp goes with it.
		assert.Equal(t, []string{"string"}, []string(props["balance"].Type))
		assert.NotContains(t, props["balance"].Extensions, "x-go-type")
	})

	t.Run("a non-body parameter and a response header", func(t *testing.T) {
		// SimpleSchema mode: `integer` and `string` are primitive, so neither needs a schema.
		params := doc.Paths.Paths["/quotes"].Get.Parameters
		byName := map[string]string{}
		for _, p := range params {
			if p.In == "query" {
				byName[p.Name] = p.Type
			}
		}
		assert.Equal(t, "integer", byName["threshold"])
		assert.Equal(t, "string", byName["tolerance"])

		assert.Equal(t, "integer", doc.Responses["quoteResponse"].Headers["X-Remaining"].Type)
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_math_big.json")
}
