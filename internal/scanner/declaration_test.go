// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestEntityDecl(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	t.Run("Obj", func(t *testing.T) {
		t.Run("named type returns TypeName", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/models",
				"User",
			)
			require.True(t, ok)
			require.NotNil(t, decl.Type)

			obj := decl.Obj()
			assert.EqualT(t, "User", obj.Name())
		})

		t.Run("panics when both Type and Alias are nil", func(t *testing.T) {
			decl := &EntityDecl{
				ident: ast.NewIdent("Bad"),
			}
			assert.Panics(t, func() {
				decl.Obj()
			})
		})
	})

	t.Run("ObjType", func(t *testing.T) {
		t.Run("named type returns types.Named", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/models",
				"User",
			)
			require.True(t, ok)

			objType := decl.ObjType()
			_, isNamed := objType.(*types.Named)
			assert.True(t, isNamed, "expected *types.Named, got %T", objType)
		})

		t.Run("alias type returns types.Alias", func(t *testing.T) {
			// Load the spec fixture which has type aliases (Customer = User).
			specCtx, err := NewScanCtx(withTestLoader(&Options{
				Packages: []string{"./goparsing/spec"},
				WorkDir:  "../../testdata",
			}))
			require.NoError(t, err)

			decl, ok := specCtx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/spec",
				"Customer",
			)
			require.True(t, ok)
			require.NotNil(t, decl.Alias, "Customer should be a type alias")

			objType := decl.ObjType()
			_, isAlias := objType.(*types.Alias)
			assert.True(t, isAlias, "expected *types.Alias, got %T", objType)
		})

		t.Run("panics when both Type and Alias are nil", func(t *testing.T) {
			decl := &EntityDecl{
				ident: ast.NewIdent("Bad"),
			}
			assert.Panics(t, func() {
				decl.ObjType()
			})
		})
	})

	t.Run("Names", func(t *testing.T) {
		t.Run("model with swagger:model annotation uses Go name", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/models",
				"User",
			)
			require.True(t, ok)

			name, goName := decl.Names()
			assert.EqualT(t, "User", goName)
			assert.EqualT(t, "User", name)
		})

		t.Run("type without model annotation returns Go name for both", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/operations",
				"SimpleOne",
			)
			require.True(t, ok)

			name, goName := decl.Names()
			assert.EqualT(t, "SimpleOne", goName)
			assert.EqualT(t, "SimpleOne", name)
		})

		t.Run("model with override name returns override", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/models",
				"BaseStruct",
			)
			require.True(t, ok)

			name, goName := decl.Names()
			assert.EqualT(t, "BaseStruct", goName)
			assert.EqualT(t, "animal", name) // override name from swagger:model animal
		})
	})

	t.Run("the type half answers without the syntax half", func(t *testing.T) {
		// A package served from compiled export data has types and positions but no parsed source.
		// Blanking the syntax half reproduces that state: everything the type half promises must
		// still answer, and none of it may panic.
		const modelsPkg = "github.com/go-openapi/codescan/testdata/goparsing/classification/models"

		found, ok := sctx.FindDecl(modelsPkg, "BaseStruct")
		require.True(t, ok)
		wantPos := sctx.PosOf(found.ident.Pos())

		decl := &EntityDecl{Type: found.Type, Alias: found.Alias}

		assert.EqualT(t, "BaseStruct", decl.Name())
		assert.EqualT(t, modelsPkg, decl.PkgPath())
		assert.EqualT(t, wantPos, sctx.PosOf(decl.Pos()))
		assert.EqualT(t, modelsPkg+"/BaseStruct", decl.DefKey())

		name, goName := decl.Names()
		assert.EqualT(t, "BaseStruct", goName)
		// The swagger:model override lives in the comments, so without them the name falls back to
		// the Go name — the one thing on this surface that genuinely needs source.
		assert.EqualT(t, "BaseStruct", name)
	})

	t.Run("the syntax half answers when there is source", func(t *testing.T) {
		const modelsPkg = "github.com/go-openapi/codescan/testdata/goparsing/classification/models"

		decl, ok := sctx.FindDecl(modelsPkg, "SomeTimedType")
		require.True(t, ok)

		assert.True(t, decl.HasSource())
		assert.NotNil(t, decl.Comments())
		assert.NotNil(t, decl.File())

		expr, ok := decl.TypeExpr()
		require.True(t, ok)
		assert.EqualT(t, decl.spec.Type, expr)

		imports, ok := decl.Imports()
		require.True(t, ok)
		assert.NotEmpty(t, imports)

		imported, ok := decl.PkgImport("github.com/go-openapi/strfmt")
		require.True(t, ok)
		assert.EqualT(t, "strfmt", imported.Name)

		_, ok = decl.PkgImport("example.com/never/imported")
		assert.False(t, ok, "a path the declaring package does not import is not an import of it")

		pkg, ok := decl.EnumSourcePkg()
		require.True(t, ok)
		assert.EqualT(t, modelsPkg, pkg.PkgPath)
	})

	t.Run("the syntax half reports its own absence", func(t *testing.T) {
		// The state a package served from compiled export data is in: types, no parsed source. Every
		// syntax accessor has to say so rather than dereference nothing.
		found, ok := sctx.FindDecl(
			"github.com/go-openapi/codescan/testdata/goparsing/classification/models",
			"SomeTimedType",
		)
		require.True(t, ok)

		decl := &EntityDecl{Type: found.Type, Alias: found.Alias}

		assert.False(t, decl.HasSource())
		assert.Nil(t, decl.Comments())
		assert.Nil(t, decl.File())

		_, ok = decl.TypeExpr()
		assert.False(t, ok)

		_, ok = decl.Imports()
		assert.False(t, ok)

		_, ok = decl.PkgImport("github.com/go-openapi/strfmt")
		assert.False(t, ok)

		_, ok = decl.EnumSourcePkg()
		assert.False(t, ok)
	})

	t.Run("Pos agrees with the declaring identifier", func(t *testing.T) {
		for _, name := range []string{"User", "BaseStruct", "SomeStringType", "SomeStringTypeAlias"} {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/models",
				name,
			)
			require.True(t, ok)

			assert.EqualT(t, decl.ident.Pos(), decl.Pos())
			assert.EqualT(t, decl.spec.Pos(), decl.Pos())
			assert.EqualT(t, decl.ident.Name, decl.Name())
		}
	})

	// Response name resolution moved to the grammar (grammar.ResponseBlock →
	// responses.Builder.ResponseName); the scanner no longer parses the swagger:response argument.
	// See the grammar parser tests and the shared-parameters response coverage.

	// Operation-id targeting parse moved to the grammar (grammar.ParametersBlock); see the grammar
	// parser tests.
	// The scanner no longer parses swagger:parameters arguments.

	t.Run("HasAnnotation caching", func(t *testing.T) {
		t.Run("HasModelAnnotation caches result", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/models",
				"User",
			)
			require.True(t, ok)

			// First call: parses comments
			assert.True(t, decl.HasModelAnnotation())
			// Second call: returns cached true
			assert.True(t, decl.HasModelAnnotation())
		})

		t.Run("HasResponseAnnotation caches result", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/operations",
				"GenericError",
			)
			require.True(t, ok)

			assert.True(t, decl.HasResponseAnnotation())
			// Second call: returns cached true
			assert.True(t, decl.HasResponseAnnotation())
		})

		t.Run("HasParameterAnnotation caches result", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/operations",
				"MyFileParams",
			)
			require.True(t, ok)

			assert.True(t, decl.HasParameterAnnotation())
			// Second call: returns cached true
			assert.True(t, decl.HasParameterAnnotation())
		})

		t.Run("HasModelAnnotation returns false for non-model", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/operations",
				"SimpleOne",
			)
			require.True(t, ok)

			assert.False(t, decl.HasModelAnnotation())
		})

		t.Run("HasResponseAnnotation returns false for non-response", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/operations",
				"SimpleOne",
			)
			require.True(t, ok)

			assert.False(t, decl.HasResponseAnnotation())
		})

		t.Run("HasParameterAnnotation returns false for non-parameter", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/testdata/goparsing/classification/operations",
				"SimpleOne",
			)
			require.True(t, ok)

			assert.False(t, decl.HasParameterAnnotation())
		})
	})
}
