// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"go/ast"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestParseRoutePathAnnotation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		line       string
		wantMethod string
		wantPath   string
		wantID     string
		wantTags   []string
	}{
		{
			name:       "GET with tags",
			line:       "// swagger:route GET /pets pets listPets",
			wantMethod: "GET",
			wantPath:   "/pets",
			wantID:     "listPets",
			wantTags:   []string{"pets"},
		},
		{
			name:       "POST without tags",
			line:       "// swagger:route POST /pets createPet",
			wantMethod: "POST",
			wantPath:   "/pets",
			wantID:     "createPet",
		},
		{
			name:       "with path params",
			line:       "// swagger:route GET /pets/{petId} pets getPet",
			wantMethod: "GET",
			wantPath:   "/pets/{petId}",
			wantID:     "getPet",
			wantTags:   []string{"pets"},
		},
		{
			name:       "multiple tags",
			line:       "// swagger:route DELETE /pets/{petId} pets admin deletePet",
			wantMethod: "DELETE",
			wantPath:   "/pets/{petId}",
			wantID:     "deletePet",
			wantTags:   []string{"pets", "admin"},
		},

		// One-character names.
		// Both regexes used to require a letter followed by AT LEAST ONE more character, and since the tags group is optional
		// the parse did not fail there — it fell back to matching with no tags, leaving the operationId pattern to swallow
		// `e listPets`, which its alphabet has no space for.
		//
		// The line then matched nothing at all, and a `swagger:route` that matches nothing is not a malformed route but
		// simply not a route: the whole annotation vanished without a word.
		// Neither restriction has any basis in OAS 2.0.
		{
			name:       "single-character tag",
			line:       "// swagger:route GET /pets e listPets",
			wantMethod: "GET",
			wantPath:   "/pets",
			wantID:     "listPets",
			wantTags:   []string{"e"},
		},
		{
			name:       "single-character operationId",
			line:       "// swagger:route GET /pets pets l",
			wantMethod: "GET",
			wantPath:   "/pets",
			wantID:     "l",
			wantTags:   []string{"pets"},
		},
		{
			name:       "single-character operationId, no tags",
			line:       "// swagger:route GET /pets l",
			wantMethod: "GET",
			wantPath:   "/pets",
			wantID:     "l",
		},
		{
			name:       "single-character tag and operationId",
			line:       "// swagger:route GET /pets e l",
			wantMethod: "GET",
			wantPath:   "/pets",
			wantID:     "l",
			wantTags:   []string{"e"},
		},
		{
			name:       "one-character tag among several",
			line:       "// swagger:route DELETE /pets/{petId} a admin deletePet",
			wantMethod: "DELETE",
			wantPath:   "/pets/{petId}",
			wantID:     "deletePet",
			wantTags:   []string{"a", "admin"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			comments := []*ast.Comment{{Text: tc.line}}
			cnt := ParseRoutePathAnnotation(comments)
			assert.EqualT(t, tc.wantMethod, cnt.Method)
			assert.EqualT(t, tc.wantPath, cnt.Path)
			assert.EqualT(t, tc.wantID, cnt.ID)
			if tc.wantTags == nil {
				assert.Nil(t, cnt.Tags)
			} else {
				assert.Equal(t, tc.wantTags, cnt.Tags)
			}
		})
	}
}

func TestParseOperationPathAnnotation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		line       string
		wantMethod string
		wantPath   string
		wantID     string
		wantTags   []string
	}{
		{
			name:       "single-character tag",
			line:       "// swagger:operation GET /v1/pets e listPets",
			wantMethod: "GET",
			wantPath:   "/v1/pets",
			wantID:     "listPets",
			wantTags:   []string{"e"},
		},
		{
			name:       "single-character operationId",
			line:       "// swagger:operation GET /v1/pets pets l",
			wantMethod: "GET",
			wantPath:   "/v1/pets",
			wantID:     "l",
			wantTags:   []string{"pets"},
		},
		{
			name:       "basic operation",
			line:       "// swagger:operation POST /v1/pets pets addPet",
			wantMethod: "POST",
			wantPath:   "/v1/pets",
			wantID:     "addPet",
			wantTags:   []string{"pets"},
		},
		{
			name:       "without tags",
			line:       "// swagger:operation GET /health checkHealth",
			wantMethod: "GET",
			wantPath:   "/health",
			wantID:     "checkHealth",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			comments := []*ast.Comment{{Text: tc.line}}
			cnt := ParseOperationPathAnnotation(comments)
			assert.EqualT(t, tc.wantMethod, cnt.Method)
			assert.EqualT(t, tc.wantPath, cnt.Path)
			assert.EqualT(t, tc.wantID, cnt.ID)
			if tc.wantTags == nil {
				assert.Nil(t, cnt.Tags)
			} else {
				assert.Equal(t, tc.wantTags, cnt.Tags)
			}
		})
	}
}

func TestParsePathAnnotation_Remaining(t *testing.T) {
	t.Parallel()

	comments := []*ast.Comment{
		{Text: "// swagger:route GET /pets pets listPets"},
		{Text: "// This is the description"},
		{Text: "// More description"},
	}

	cnt := ParseRoutePathAnnotation(comments)
	assert.EqualT(t, "GET", cnt.Method)
	require.NotNil(t, cnt.Remaining)
	assert.Len(t, cnt.Remaining.List, 2)
}

func TestParsePathAnnotation_NoMatch(t *testing.T) {
	t.Parallel()

	comments := []*ast.Comment{
		{Text: "// just a regular comment"},
	}

	cnt := ParseRoutePathAnnotation(comments)
	assert.EqualT(t, "", cnt.Method)
	assert.EqualT(t, "", cnt.Path)
	assert.EqualT(t, "", cnt.ID)
}

func TestStripPathParamRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		in           string
		wantCleaned  string
		wantStripped []string
	}{
		{name: "no params", in: "/items", wantCleaned: "/items"},
		{name: "plain template untouched", in: "/items/{id}", wantCleaned: "/items/{id}"},
		{
			name: "simple regex", in: "/items/{id:[0-9]+}",
			wantCleaned: "/items/{id}", wantStripped: []string{"id"},
		},
		{
			name: "regex with slash class", in: "/files/{path:[^/]+}",
			wantCleaned: "/files/{path}", wantStripped: []string{"path"},
		},
		{
			name: "nested-brace quantifier", in: "/codes/{code:[0-9]{2,4}}",
			wantCleaned: "/codes/{code}", wantStripped: []string{"code"},
		},
		{
			name: "multiple params, mixed", in: "/a/{x:[0-9]+}/b/{y}/c/{z:[a-z]+}",
			wantCleaned: "/a/{x}/b/{y}/c/{z}", wantStripped: []string{"x", "z"},
		},
		{
			name: "whole route line", in: "// swagger:route GET /items/{id:[0-9]+} items getItem",
			wantCleaned: "// swagger:route GET /items/{id} items getItem", wantStripped: []string{"id"},
		},
		{
			name: "unbalanced brace left verbatim", in: "/items/{id:[0-9]+",
			wantCleaned: "/items/{id:[0-9]+",
		},
		{
			name: "empty name with colon left verbatim", in: "/items/{:[0-9]+}",
			wantCleaned: "/items/{:[0-9]+}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleaned, stripped := stripPathParamRegex(tc.in)
			assert.EqualT(t, tc.wantCleaned, cleaned)
			if tc.wantStripped == nil {
				assert.Nil(t, stripped)
			} else {
				assert.Equal(t, tc.wantStripped, stripped)
			}
		})
	}
}

func TestParseRoutePathAnnotation_RegexSegments(t *testing.T) {
	t.Parallel()

	comments := []*ast.Comment{
		{Text: "// swagger:route GET /a/{x:[0-9]+}/b/{y:[a-z]+} items getThings"},
	}
	cnt := ParseRoutePathAnnotation(comments)
	assert.EqualT(t, "GET", cnt.Method)
	assert.EqualT(t, "/a/{x}/b/{y}", cnt.Path)
	assert.EqualT(t, "getThings", cnt.ID)
	assert.Equal(t, []string{"x", "y"}, cnt.StrippedParams)
}
