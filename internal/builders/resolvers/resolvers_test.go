// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"errors"
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/go-openapi/codescan/internal/ifaces"
	"github.com/go-openapi/codescan/internal/scantest/mocks"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestAddExtension(t *testing.T) {
	ve := &oaispec.VendorExtensible{
		Extensions: make(oaispec.Extensions),
	}

	key := "x-go-name"
	value := "Name"
	AddExtension(ve, key, value, false)
	veStr, ok := ve.Extensions[key].(string)
	require.TrueT(t, ok)
	assert.EqualT(t, value, veStr)

	key2 := "x-go-package"
	value2 := "schema"
	AddExtension(ve, key2, value2, false)
	veStr2, ok := ve.Extensions[key2].(string)
	require.TrueT(t, ok)
	assert.EqualT(t, value2, veStr2)

	key3 := "x-go-class"
	value3 := "Spec"
	AddExtension(ve, key3, value3, true)
	assert.Nil(t, ve.Extensions[key3])
}

// typableCapture builds a MockSwaggerTypable whose Typed() records
// the (swaggerType, format) pair it receives.
func typableCapture() (*mocks.MockSwaggerTypable, *[2]string) {
	got := new([2]string)
	m := &mocks.MockSwaggerTypable{
		TypedFunc: func(swaggerType, format string) {
			got[0] = swaggerType
			got[1] = format
		},
	}
	return m, got
}

func TestSwaggerSchemaForType(t *testing.T) {
	t.Run("supported builtins map to swagger types", func(t *testing.T) {
		cases := []struct {
			in   string
			want [2]string
		}{
			{"bool", [2]string{"boolean", ""}},
			{"byte", [2]string{"integer", "uint8"}},
			{"error", [2]string{"string", ""}},
			{"float32", [2]string{"number", "float"}},
			{"float64", [2]string{"number", "double"}},
			{"int", [2]string{"integer", "int64"}},
			{"int8", [2]string{"integer", "int8"}},
			{"int16", [2]string{"integer", "int16"}},
			{"int32", [2]string{"integer", "int32"}},
			{"int64", [2]string{"integer", "int64"}},
			{"rune", [2]string{"integer", "int32"}},
			{"string", [2]string{"string", ""}},
			{"uint", [2]string{"integer", "uint64"}},
			{"uint8", [2]string{"integer", "uint8"}},
			{"uint16", [2]string{"integer", "uint16"}},
			{"uint32", [2]string{"integer", "uint32"}},
			{"uint64", [2]string{"integer", "uint64"}},
			{"uintptr", [2]string{"integer", "uint64"}},
			{"object", [2]string{"object", ""}},
		}

		for _, tc := range cases {
			t.Run(tc.in, func(t *testing.T) {
				m, got := typableCapture()
				require.NoError(t, SwaggerSchemaForType(tc.in, m))
				assert.EqualT(t, tc.want, *got)
			})
		}
	})

	t.Run("complex64/128 returns ErrResolver", func(t *testing.T) {
		for _, name := range []string{"complex64", "complex128"} {
			t.Run(name, func(t *testing.T) {
				m := &mocks.MockSwaggerTypable{
					TypedFunc: func(_, _ string) { t.Fatalf("Typed should not be called for %q", name) },
				}
				err := SwaggerSchemaForType(name, m)
				require.Error(t, err)
				require.TrueT(t, errors.Is(err, ErrResolver))
			})
		}
	})

	t.Run("unknown type returns ErrResolver", func(t *testing.T) {
		m := &mocks.MockSwaggerTypable{
			TypedFunc: func(_, _ string) { t.Fatal("Typed should not be called for unknown type") },
		}
		err := SwaggerSchemaForType("bogus", m)
		require.Error(t, err)
		require.TrueT(t, errors.Is(err, ErrResolver))
	})
}

func TestUnsupportedBuiltin(t *testing.T) {
	t.Run("nil Obj returns false", func(t *testing.T) {
		m := &mocks.MockObjecter{ObjFunc: func() *types.TypeName { return nil }}
		assert.FalseT(t, UnsupportedBuiltin(m))
	})

	t.Run("unsafe package returns true", func(t *testing.T) {
		pkg := types.NewPackage("unsafe", "unsafe")
		tn := types.NewTypeName(token.NoPos, pkg, "Pointer", nil)
		m := &mocks.MockObjecter{ObjFunc: func() *types.TypeName { return tn }}
		assert.TrueT(t, UnsupportedBuiltin(m))
	})

	t.Run("user package returns false", func(t *testing.T) {
		pkg := types.NewPackage("example.com/foo", "foo")
		tn := types.NewTypeName(token.NoPos, pkg, "Bar", nil)
		m := &mocks.MockObjecter{ObjFunc: func() *types.TypeName { return tn }}
		assert.FalseT(t, UnsupportedBuiltin(m))
	})

	t.Run("universe-scope object returns false", func(t *testing.T) {
		// Pkg()==nil objects (predeclared error, any, etc.) never
		// represent unsafe.Pointer and must therefore not match.
		tn := types.NewTypeName(token.NoPos, nil, "error", nil)
		m := &mocks.MockObjecter{ObjFunc: func() *types.TypeName { return tn }}
		assert.FalseT(t, UnsupportedBuiltin(m))
	})
}

func TestUnsupportedBuiltinType(t *testing.T) {
	t.Run("Basic complex64 returns true", func(t *testing.T) {
		assert.TrueT(t, UnsupportedBuiltinType(types.Typ[types.Complex64]))
	})

	t.Run("Basic int returns false", func(t *testing.T) {
		assert.FalseT(t, UnsupportedBuiltinType(types.Typ[types.Int]))
	})

	t.Run("UnsafePointer Basic returns true", func(t *testing.T) {
		assert.TrueT(t, UnsupportedBuiltinType(types.Typ[types.UnsafePointer]))
	})

	t.Run("Chan returns true", func(t *testing.T) {
		ch := types.NewChan(types.SendRecv, types.Typ[types.Int])
		assert.TrueT(t, UnsupportedBuiltinType(ch))
	})

	t.Run("Signature returns true", func(t *testing.T) {
		sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
		assert.TrueT(t, UnsupportedBuiltinType(sig))
	})

	t.Run("TypeParam returns true", func(t *testing.T) {
		pkg := types.NewPackage("example.com/foo", "foo")
		tn := types.NewTypeName(token.NoPos, pkg, "T", nil)
		tp := types.NewTypeParam(tn, types.NewInterfaceType(nil, nil))
		assert.TrueT(t, UnsupportedBuiltinType(tp))
	})

	t.Run("Named delegates to UnsupportedBuiltin", func(t *testing.T) {
		pkg := types.NewPackage("example.com/foo", "foo")
		tn := types.NewTypeName(token.NoPos, pkg, "Foo", nil)
		named := types.NewNamed(tn, types.Typ[types.Int], nil)
		assert.FalseT(t, UnsupportedBuiltinType(named))
	})

	t.Run("Map (default case) returns false", func(t *testing.T) {
		m := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
		assert.FalseT(t, UnsupportedBuiltinType(m))
	})
}

func TestIsFieldStringable(t *testing.T) {
	t.Run("scalar Ident returns true", func(t *testing.T) {
		for _, name := range []string{"int", "int8", "int64", "uint32", "float64", "string", "bool"} {
			assert.TrueT(t, IsFieldStringable(ast.NewIdent(name)), "want true for %s", name)
		}
	})

	t.Run("non-scalar Ident returns false", func(t *testing.T) {
		for _, name := range []string{"any", "Foo", "float32"} { // float32 isn't in the stringable set
			assert.FalseT(t, IsFieldStringable(ast.NewIdent(name)), "want false for %s", name)
		}
	})

	t.Run("StarExpr to scalar returns true", func(t *testing.T) {
		star := &ast.StarExpr{X: ast.NewIdent("int")}
		assert.TrueT(t, IsFieldStringable(star))
	})

	t.Run("StarExpr to non-scalar returns false", func(t *testing.T) {
		star := &ast.StarExpr{X: ast.NewIdent("Foo")}
		assert.FalseT(t, IsFieldStringable(star))
	})

	t.Run("other AST expr returns false", func(t *testing.T) {
		// SelectorExpr like pkg.Type — neither Ident nor StarExpr.
		sel := &ast.SelectorExpr{X: ast.NewIdent("pkg"), Sel: ast.NewIdent("Type")}
		assert.FalseT(t, IsFieldStringable(sel))

		// ArrayType — same, hits the else-return-false branch.
		arr := &ast.ArrayType{Elt: ast.NewIdent("int")}
		assert.FalseT(t, IsFieldStringable(arr))
	})
}

func TestParseJSONTag(t *testing.T) {
	ident := func(name string) *ast.Field {
		return &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(name)},
			Type:  ast.NewIdent("int"),
		}
	}

	multi := func(names ...string) *ast.Field {
		idents := make([]*ast.Ident, 0, len(names))
		for _, n := range names {
			idents = append(idents, ast.NewIdent(n))
		}
		return &ast.Field{Names: idents, Type: ast.NewIdent("int")}
	}

	t.Run("no tag uses goName", func(t *testing.T) {
		f := ident("Foo")
		name, ignore, isString, omitEmpty, err := ParseJSONTag(f, "Foo")
		require.NoError(t, err)
		assert.EqualT(t, "Foo", name)
		assert.FalseT(t, ignore)
		assert.FalseT(t, isString)
		assert.FalseT(t, omitEmpty)
	})

	t.Run("empty goName falls back to first AST name", func(t *testing.T) {
		f := ident("Foo")
		name, _, _, _, err := ParseJSONTag(f, "")
		require.NoError(t, err)
		assert.EqualT(t, "Foo", name)
	})

	t.Run("json tag renames", func(t *testing.T) {
		f := ident("Foo")
		f.Tag = &ast.BasicLit{Value: "`json:\"foo,omitempty\"`"}
		name, ignore, isString, omitEmpty, err := ParseJSONTag(f, "Foo")
		require.NoError(t, err)
		assert.EqualT(t, "foo", name)
		assert.FalseT(t, ignore)
		assert.FalseT(t, isString)
		assert.TrueT(t, omitEmpty)
	})

	t.Run("json:\"-\" marks ignored", func(t *testing.T) {
		f := ident("Foo")
		f.Tag = &ast.BasicLit{Value: "`json:\"-\"`"}
		name, ignore, _, _, err := ParseJSONTag(f, "Foo")
		require.NoError(t, err)
		assert.EqualT(t, "Foo", name)
		assert.TrueT(t, ignore)
	})

	t.Run("json:\",string\" on scalar sets isString", func(t *testing.T) {
		f := ident("Foo")
		f.Tag = &ast.BasicLit{Value: "`json:\",string\"`"}
		name, _, isString, _, err := ParseJSONTag(f, "Foo")
		require.NoError(t, err)
		assert.EqualT(t, "Foo", name)
		assert.TrueT(t, isString)
	})

	t.Run("whitespace-only tag value falls through", func(t *testing.T) {
		f := ident("Foo")
		// Backticks wrap a single space — TrimSpace of `" "` != empty (outer check
		// passes), but Unquote yields " " which does TrimSpace to empty — hits
		// the final fallthrough.
		f.Tag = &ast.BasicLit{Value: "` `"}
		name, ignore, isString, omitEmpty, err := ParseJSONTag(f, "Foo")
		require.NoError(t, err)
		assert.EqualT(t, "Foo", name)
		assert.FalseT(t, ignore)
		assert.FalseT(t, isString)
		assert.FalseT(t, omitEmpty)
	})

	t.Run("malformed tag returns Unquote error", func(t *testing.T) {
		f := ident("Foo")
		// Unquote requires surrounding backticks/quotes. Bare word is invalid.
		f.Tag = &ast.BasicLit{Value: "not-a-quoted-tag"}
		_, _, _, _, err := ParseJSONTag(f, "Foo")
		require.Error(t, err)
	})

	t.Run("multi-name group keeps each goName (go-swagger#2638)", func(t *testing.T) {
		// `R, G, B, A uint8` — go/types yields one var per name sharing the
		// same AST field; each member must keep its own name.
		f := multi("R", "G", "B", "A")
		for _, n := range []string{"R", "G", "B", "A"} {
			name, _, _, _, err := ParseJSONTag(f, n)
			require.NoError(t, err)
			assert.EqualT(t, n, name)
		}
	})

	t.Run("rename ignored on multi-name group, options still apply", func(t *testing.T) {
		f := multi("R", "G")
		f.Tag = &ast.BasicLit{Value: "`json:\"renamed,omitempty\"`"}
		name, ignore, _, omitEmpty, err := ParseJSONTag(f, "G")
		require.NoError(t, err)
		assert.EqualT(t, "G", name, "a single rename cannot name N members")
		assert.FalseT(t, ignore)
		assert.TrueT(t, omitEmpty, "omitempty still applies to every member")
	})

	t.Run("json:\"-\" on multi-name group ignores all members", func(t *testing.T) {
		f := multi("R", "G")
		f.Tag = &ast.BasicLit{Value: "`json:\"-\"`"}
		_, ignore, _, _, err := ParseJSONTag(f, "R")
		require.NoError(t, err)
		assert.TrueT(t, ignore)
	})
}

func TestMustNotBeABuiltinType(t *testing.T) {
	t.Run("user type does not panic", func(t *testing.T) {
		pkg := types.NewPackage("example.com/foo", "foo")
		tn := types.NewTypeName(token.NoPos, pkg, "Foo", nil)
		assert.NotPanics(t, func() { MustNotBeABuiltinType(tn) })
	})

	t.Run("builtin panics wrapping ErrInternal", func(t *testing.T) {
		tn := types.NewTypeName(token.NoPos, nil, "complex64", nil)
		defer func() {
			r := recover()
			require.NotNil(t, r)
			err, ok := r.(error)
			require.TrueT(t, ok, "panic value should be an error")
			require.TrueT(t, errors.Is(err, ErrInternal))
		}()
		MustNotBeABuiltinType(tn)
	})
}

func TestInternalError_Error(t *testing.T) {
	// Exercises the internalError.Error method that satisfies the `error`
	// interface used in fmt.Errorf("...%w", ErrInternal).
	assert.EqualT(t, string(ErrInternal), ErrInternal.Error())
}

// Assert at compile time that our typable fixture satisfies the interface.
var _ ifaces.SwaggerTypable = (*mocks.MockSwaggerTypable)(nil)
