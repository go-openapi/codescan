// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"go/ast"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan/internal/ifaces"
	oaispec "github.com/go-openapi/spec"
)

// stubTypable is a minimal SwaggerTypable for testing IsAliasParam.
type stubTypable struct {
	in string
}

func (s stubTypable) In() string           { return s.in }
func (s stubTypable) Typed(string, string) {}
func (s stubTypable) SetRef(oaispec.Ref)   {}

func (s stubTypable) Items() ifaces.SwaggerTypable { //nolint:ireturn // test stub
	return s
}
func (s stubTypable) Schema() *oaispec.Schema    { return nil }
func (s stubTypable) Level() int                 { return 0 }
func (s stubTypable) AddExtension(string, any)   {}
func (s stubTypable) WithEnum(...any)            {}
func (s stubTypable) WithEnumDescription(string) {}

func TestHasAnnotation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want bool
	}{
		{"swagger:model", "// swagger:model Foo", true},
		{"swagger:route", "swagger:route GET /foo tags fooOp", true},
		{"swagger:parameters", "// swagger:parameters addFoo", true},
		{"swagger:response", "swagger:response notFound", true},
		{"swagger:operation", "// swagger:operation POST /bar tags barOp", true},
		{"no annotation", "// this is just a comment", false},
		{"empty", "", false},
		{"partial", "swagger:", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.EqualT(t, tc.want, HasAnnotation(tc.line))
		})
	}
}

func TestIsAliasParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want bool
	}{
		{"query", true},
		{"path", true},
		{"formData", true},
		{"body", false},
		{"header", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.EqualT(t, tc.want, IsAliasParam(stubTypable{in: tc.in}))
		})
	}
}

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

func TestCommentMatcher(t *testing.T) {
	t.Parallel()

	t.Run("AllOfMember", func(t *testing.T) {
		assert.TrueT(t, AllOfMember(makeCommentGroup("// swagger:allOf")))
		assert.TrueT(t, AllOfMember(makeCommentGroup("// swagger:allOf MyParent")))
		assert.FalseT(t, AllOfMember(makeCommentGroup("// just a comment")))
		assert.FalseT(t, AllOfMember(nil))
	})

	t.Run("FileParam", func(t *testing.T) {
		assert.TrueT(t, FileParam(makeCommentGroup("// swagger:file")))
		assert.FalseT(t, FileParam(makeCommentGroup("// swagger:model")))
		assert.FalseT(t, FileParam(nil))
	})

	t.Run("Ignored", func(t *testing.T) {
		assert.TrueT(t, Ignored(makeCommentGroup("// swagger:ignore")))
		assert.TrueT(t, Ignored(makeCommentGroup("// swagger:ignore Foo")))
		assert.FalseT(t, Ignored(makeCommentGroup("// swagger:model")))
		assert.FalseT(t, Ignored(nil))
	})

	t.Run("AliasParam", func(t *testing.T) {
		assert.TrueT(t, AliasParam(makeCommentGroup("// swagger:alias")))
		assert.FalseT(t, AliasParam(makeCommentGroup("// swagger:model")))
		assert.FalseT(t, AliasParam(nil))
	})
}

func TestCommentSubMatcher(t *testing.T) {
	t.Parallel()

	t.Run("StrfmtName", func(t *testing.T) {
		name, ok := StrfmtName(makeCommentGroup("// swagger:strfmt date-time"))
		require.TrueT(t, ok)
		assert.EqualT(t, "date-time", name)

		_, ok = StrfmtName(makeCommentGroup("// swagger:model Foo"))
		assert.FalseT(t, ok)

		_, ok = StrfmtName(nil)
		assert.FalseT(t, ok)
	})

	t.Run("ParamLocation", func(t *testing.T) {
		loc, ok := ParamLocation(makeCommentGroup("// In: query"))
		require.TrueT(t, ok)
		assert.EqualT(t, "query", loc)

		loc, ok = ParamLocation(makeCommentGroup("// in: body"))
		require.TrueT(t, ok)
		assert.EqualT(t, "body", loc)

		_, ok = ParamLocation(makeCommentGroup("// no location"))
		assert.FalseT(t, ok)
	})

	t.Run("EnumName", func(t *testing.T) {
		name, ok := EnumName(makeCommentGroup("// swagger:enum Status"))
		require.TrueT(t, ok)
		assert.EqualT(t, "Status", name)

		_, ok = EnumName(nil)
		assert.FalseT(t, ok)
	})

	t.Run("AllOfName", func(t *testing.T) {
		name, ok := AllOfName(makeCommentGroup("// swagger:allOf MyParent"))
		require.TrueT(t, ok)
		assert.EqualT(t, "MyParent", name)

		// bare annotation: no submatch → returns false
		_, ok = AllOfName(makeCommentGroup("// swagger:allOf"))
		assert.FalseT(t, ok)
	})

	t.Run("NameOverride", func(t *testing.T) {
		name, ok := NameOverride(makeCommentGroup("// swagger:name MyName"))
		require.TrueT(t, ok)
		assert.EqualT(t, "MyName", name)

		_, ok = NameOverride(nil)
		assert.FalseT(t, ok)
	})

	t.Run("DefaultName", func(t *testing.T) {
		name, ok := DefaultName(makeCommentGroup("// swagger:default MyDefault"))
		require.TrueT(t, ok)
		assert.EqualT(t, "MyDefault", name)
	})

	t.Run("TypeName", func(t *testing.T) {
		name, ok := TypeName(makeCommentGroup("// swagger:type string"))
		require.TrueT(t, ok)
		assert.EqualT(t, "string", name)
	})
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
