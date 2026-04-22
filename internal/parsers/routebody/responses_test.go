// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package routebody

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

func TestSetOpResponses_Matches(t *testing.T) {
	t.Parallel()

	sr := NewSetResponses(nil, nil, nil)
	assert.TrueT(t, sr.Matches("responses:"))
	assert.TrueT(t, sr.Matches("Responses:"))
	assert.FalseT(t, sr.Matches("something else"))
}

func TestSetOpResponses_Parse(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		var called bool
		sr := NewSetResponses(nil, nil, func(_ *oaispec.Response, _ map[int]oaispec.Response) { called = true })
		require.NoError(t, sr.Parse(nil))
		assert.FalseT(t, called)
		require.NoError(t, sr.Parse([]string{}))
		require.NoError(t, sr.Parse([]string{""}))
	})

	t.Run("response ref", func(t *testing.T) {
		responses := map[string]oaispec.Response{
			"notFound": {ResponseProps: oaispec.ResponseProps{Description: "not found"}},
		}
		var gotDef *oaispec.Response
		var gotScr map[int]oaispec.Response

		sr := NewSetResponses(nil, responses, func(def *oaispec.Response, scr map[int]oaispec.Response) {
			gotDef = def
			gotScr = scr
		})

		require.NoError(t, sr.Parse([]string{"404: notFound"}))
		assert.Nil(t, gotDef)
		require.NotNil(t, gotScr)
		resp, ok := gotScr[404]
		require.TrueT(t, ok)
		assert.NotEmpty(t, resp.Ref.String())
	})

	t.Run("default response", func(t *testing.T) {
		responses := map[string]oaispec.Response{
			"genericError": {},
		}
		var gotDef *oaispec.Response

		sr := NewSetResponses(nil, responses, func(def *oaispec.Response, _ map[int]oaispec.Response) {
			gotDef = def
		})

		require.NoError(t, sr.Parse([]string{"default: genericError"}))
		require.NotNil(t, gotDef)
	})

	t.Run("body ref", func(t *testing.T) {
		definitions := map[string]oaispec.Schema{
			"Pet": {},
		}
		var gotScr map[int]oaispec.Response

		sr := NewSetResponses(definitions, nil, func(_ *oaispec.Response, scr map[int]oaispec.Response) {
			gotScr = scr
		})

		require.NoError(t, sr.Parse([]string{"200: body:Pet"}))
		require.NotNil(t, gotScr)
		resp := gotScr[200]
		require.NotNil(t, resp.Schema)
		assert.Contains(t, resp.Schema.Ref.String(), "definitions/Pet")
	})

	t.Run("body array ref", func(t *testing.T) {
		definitions := map[string]oaispec.Schema{
			"Pet": {},
		}
		var gotScr map[int]oaispec.Response

		sr := NewSetResponses(definitions, nil, func(_ *oaispec.Response, scr map[int]oaispec.Response) {
			gotScr = scr
		})

		require.NoError(t, sr.Parse([]string{"200: body:[]Pet"}))
		require.NotNil(t, gotScr)
		resp := gotScr[200]
		require.NotNil(t, resp.Schema)
		assert.EqualT(t, "array", resp.Schema.Type[0])
		require.NotNil(t, resp.Schema.Items)
		assert.Contains(t, resp.Schema.Items.Schema.Ref.String(), "definitions/Pet")
	})

	t.Run("with description tag", func(t *testing.T) {
		responses := map[string]oaispec.Response{
			"notFound": {},
		}
		var gotScr map[int]oaispec.Response

		sr := NewSetResponses(nil, responses, func(_ *oaispec.Response, scr map[int]oaispec.Response) {
			gotScr = scr
		})

		require.NoError(t, sr.Parse([]string{"404: response:notFound description:Not Found"}))
		require.NotNil(t, gotScr)
		assert.EqualT(t, "Not Found", gotScr[404].Description)
	})
}

func TestParseTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		line         string
		wantModel    string
		wantArrays   int
		wantIsDefRef bool
		wantDesc     string
		wantErr      bool
	}{
		{"response ref", "notFound", "notFound", 0, false, "", false},
		{"body ref", "body:Pet", "Pet", 0, true, "", false},
		{"body array", "body:[]Pet", "Pet", 1, true, "", false},
		{"body nested array", "body:[][]Pet", "Pet", 2, true, "", false},
		{"with description", "notFound description:Resource not found", "notFound", 0, false, "Resource not found", false},
		{"invalid tag", "invalid:tag value:wrong", "", 0, false, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model, arrays, isDefRef, desc, err := parseTags(tc.line)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.EqualT(t, tc.wantModel, model)
			assert.EqualT(t, tc.wantArrays, arrays)
			assert.EqualT(t, tc.wantIsDefRef, isDefRef)
			assert.EqualT(t, tc.wantDesc, desc)
		})
	}
}

func TestAssignResponse(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		resp := oaispec.Response{ResponseProps: oaispec.ResponseProps{Description: "error"}}
		def, scr := assignResponse("default", resp, nil, nil)
		require.NotNil(t, def)
		assert.EqualT(t, "error", def.Description)
		assert.Nil(t, scr)
	})

	t.Run("default already set", func(t *testing.T) {
		existing := &oaispec.Response{ResponseProps: oaispec.ResponseProps{Description: "existing"}}
		def, _ := assignResponse("default", oaispec.Response{}, existing, nil)
		assert.EqualT(t, "existing", def.Description) // not overwritten
	})

	t.Run("status code", func(t *testing.T) {
		resp := oaispec.Response{ResponseProps: oaispec.ResponseProps{Description: "ok"}}
		def, scr := assignResponse("200", resp, nil, nil)
		assert.Nil(t, def)
		require.NotNil(t, scr)
		assert.EqualT(t, "ok", scr[200].Description)
	})

	t.Run("non-numeric key ignored", func(t *testing.T) {
		def, scr := assignResponse("notANumber", oaispec.Response{}, nil, nil)
		assert.Nil(t, def)
		assert.Nil(t, scr)
	})
}

func TestSetOpResponses_ParseEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("line without colon", func(t *testing.T) {
		var gotScr map[int]oaispec.Response
		sr := NewSetResponses(nil, nil, func(_ *oaispec.Response, scr map[int]oaispec.Response) {
			gotScr = scr
		})
		require.NoError(t, sr.Parse([]string{"no-colon-here"}))
		assert.Nil(t, gotScr)
	})

	t.Run("empty key after trim", func(t *testing.T) {
		var gotScr map[int]oaispec.Response
		sr := NewSetResponses(nil, nil, func(_ *oaispec.Response, scr map[int]oaispec.Response) {
			gotScr = scr
		})
		require.NoError(t, sr.Parse([]string{" : someValue"}))
		assert.Nil(t, gotScr)
	})

	t.Run("empty value assigns empty response", func(t *testing.T) {
		var gotScr map[int]oaispec.Response
		sr := NewSetResponses(nil, nil, func(_ *oaispec.Response, scr map[int]oaispec.Response) {
			gotScr = scr
		})
		require.NoError(t, sr.Parse([]string{"200:"}))
		require.NotNil(t, gotScr)
		_, ok := gotScr[200]
		assert.TrueT(t, ok)
	})

	t.Run("parse error propagated", func(t *testing.T) {
		sr := NewSetResponses(nil, nil, func(_ *oaispec.Response, _ map[int]oaispec.Response) {})
		// "invalid:tag" is not response/body/description → error
		err := sr.Parse([]string{"200: invalid:tag"})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrParser)
	})

	t.Run("definition found by fallback lookup", func(t *testing.T) {
		// refTarget is not in responses but IS in definitions → isDefinitionRef becomes true
		definitions := map[string]oaispec.Schema{
			"ErrorModel": {},
		}
		var gotScr map[int]oaispec.Response
		sr := NewSetResponses(definitions, nil, func(_ *oaispec.Response, scr map[int]oaispec.Response) {
			gotScr = scr
		})
		require.NoError(t, sr.Parse([]string{"500: ErrorModel"}))
		require.NotNil(t, gotScr)
		resp := gotScr[500]
		require.NotNil(t, resp.Schema)
		assert.Contains(t, resp.Schema.Ref.String(), "definitions/ErrorModel")
	})
}

func TestParseTags_UntaggedValues(t *testing.T) {
	t.Parallel()

	t.Run("second value defaults to description tag", func(t *testing.T) {
		// "notFound Something" → first untagged = responseTag, second untagged = descriptionTag
		model, _, _, desc, err := parseTags("notFound Something here")
		require.NoError(t, err)
		assert.EqualT(t, "notFound", model)
		assert.EqualT(t, "Something here", desc)
	})

	t.Run("response tag out of position", func(t *testing.T) {
		// response: after first value already parsed
		_, _, _, _, err := parseTags("body:Pet response:duplicate")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrParser)
	})
}
