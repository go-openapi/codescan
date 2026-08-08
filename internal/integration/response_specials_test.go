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

// A `swagger:response` declared directly on a recognized stdlib type renders the same in both
// spellings a Go author can reach it by.
//
// The recognizers answer from the object alone, so they run ahead of the underlying-shape dispatch.
// Putting them after it lost every type that is a struct underneath: time.Time went to the struct
// arm to have its fields read as response headers, exports none, and the response carried no schema.
func TestResponseSpecials(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/response-specials/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	t.Run("every declared response carries a schema", func(t *testing.T) {
		// The defect was silent: no error, no diagnostic, just a response with a description and
		// nothing else.
		for name, resp := range doc.Responses {
			assert.NotNil(t, resp.Schema, "%s must carry a schema", name)
		}
	})

	t.Run("the two spellings agree", func(t *testing.T) {
		for defined, aliased := range map[string]string{
			"definedStamp":  "aliasedStamp",
			"definedRaw":    "aliasedRaw",
			"definedStream": "aliasedStream",
			"definedWhole":  "aliasedWhole",
		} {
			d, a := doc.Responses[defined].Schema, doc.Responses[aliased].Schema
			require.NotNil(t, d)
			require.NotNil(t, a)
			assert.Equal(t, d.Type, a.Type, "%s and %s are the same Go type", defined, aliased)
			assert.Equal(t, d.Format, a.Format, "%s and %s are the same Go type", defined, aliased)
		}
	})

	t.Run("each type keeps its own answer", func(t *testing.T) {
		for name, wantType := range map[string]string{
			"aliasedStamp":    "string",
			"aliasedStream":   "string",
			"aliasedWhole":    "integer",
			"aliasedFraction": "string",
		} {
			sch := doc.Responses[name].Schema
			require.NotNil(t, sch)
			require.Len(t, sch.Type, 1, "%s must carry exactly one type", name)
			assert.Equal(t, wantType, sch.Type[0], "%s", name)
		}

		assert.Equal(t, "date-time", doc.Responses["aliasedStamp"].Schema.Format)
		assert.Equal(t, "byte", doc.Responses["aliasedStream"].Schema.Format)

		// json.RawMessage is "any JSON", so an empty schema is the answer, not a missing one.
		raw := doc.Responses["aliasedRaw"].Schema
		require.NotNil(t, raw)
		assert.Empty(t, raw.Type)
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_response_specials.json")
}
