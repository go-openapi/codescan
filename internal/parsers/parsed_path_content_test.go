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
