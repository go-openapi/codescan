// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"go/ast"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestExtractAnnotation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		line   string
		want   string
		wantOK bool
	}{
		{"model", "// swagger:model Foo", "model", true},
		{"route", "swagger:route GET /foo tags fooOp", "route", true},
		{"parameters", "// swagger:parameters addFoo", "parameters", true},
		{"no annotation", "// just a comment", "", false},
		{"empty", "", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ExtractAnnotation(tc.line)
			assert.EqualT(t, tc.wantOK, ok)
			assert.EqualT(t, tc.want, got)
		})
	}
}

// makeCommentGroup builds an *ast.CommentGroup from raw lines.
func makeCommentGroup(lines ...string) *ast.CommentGroup {
	if len(lines) == 0 {
		return nil
	}
	cg := &ast.CommentGroup{}
	for _, line := range lines {
		cg.List = append(cg.List, &ast.Comment{Text: line})
	}
	return cg
}

func TestCommentBlankSubMatcher(t *testing.T) {
	t.Parallel()

	t.Run("ModelOverride with name", func(t *testing.T) {
		name, ok := ModelOverride(makeCommentGroup("// swagger:model MyModel"))
		require.TrueT(t, ok)
		assert.EqualT(t, "MyModel", name)
	})

	t.Run("ModelOverride bare", func(t *testing.T) {
		name, ok := ModelOverride(makeCommentGroup("// swagger:model"))
		assert.TrueT(t, ok) // bare annotation is recognized
		assert.EqualT(t, "", name)
	})

	t.Run("ModelOverride nil", func(t *testing.T) {
		_, ok := ModelOverride(nil)
		assert.FalseT(t, ok)
	})

	t.Run("ResponseOverride with name", func(t *testing.T) {
		name, ok := ResponseOverride(makeCommentGroup("// swagger:response notFound"))
		require.TrueT(t, ok)
		assert.EqualT(t, "notFound", name)
	})

	t.Run("ResponseOverride bare", func(t *testing.T) {
		name, ok := ResponseOverride(makeCommentGroup("// swagger:response"))
		assert.TrueT(t, ok)
		assert.EqualT(t, "", name)
	})
}

func TestCommentMultipleSubMatcher(t *testing.T) {
	t.Parallel()

	t.Run("ParametersOverride single", func(t *testing.T) {
		names, ok := ParametersOverride(makeCommentGroup("// swagger:parameters addFoo"))
		require.TrueT(t, ok)
		assert.Equal(t, []string{"addFoo"}, names)
	})

	t.Run("ParametersOverride multiple", func(t *testing.T) {
		names, ok := ParametersOverride(makeCommentGroup(
			"// swagger:parameters addFoo",
			"// swagger:parameters updateBar",
		))
		require.TrueT(t, ok)
		assert.Equal(t, []string{"addFoo", "updateBar"}, names)
	})

	t.Run("ParametersOverride nil", func(t *testing.T) {
		_, ok := ParametersOverride(nil)
		assert.FalseT(t, ok)
	})

	t.Run("ParametersOverride no match", func(t *testing.T) {
		_, ok := ParametersOverride(makeCommentGroup("// just a comment"))
		assert.FalseT(t, ok)
	})
}
