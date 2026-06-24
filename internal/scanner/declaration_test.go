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
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
				"User",
			)
			require.True(t, ok)
			require.NotNil(t, decl.Type)

			obj := decl.Obj()
			assert.EqualT(t, "User", obj.Name())
		})

		t.Run("panics when both Type and Alias are nil", func(t *testing.T) {
			decl := &EntityDecl{
				Ident: ast.NewIdent("Bad"),
			}
			assert.Panics(t, func() {
				decl.Obj()
			})
		})
	})

	t.Run("ObjType", func(t *testing.T) {
		t.Run("named type returns types.Named", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
				"User",
			)
			require.True(t, ok)

			objType := decl.ObjType()
			_, isNamed := objType.(*types.Named)
			assert.True(t, isNamed, "expected *types.Named, got %T", objType)
		})

		t.Run("alias type returns types.Alias", func(t *testing.T) {
			// Load the spec fixture which has type aliases (Customer = User).
			specCtx, err := NewScanCtx(&Options{
				Packages: []string{"./goparsing/spec"},
				WorkDir:  "../../fixtures",
			})
			require.NoError(t, err)

			decl, ok := specCtx.FindDecl(
				"github.com/go-openapi/codescan/fixtures/goparsing/spec",
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
				Ident: ast.NewIdent("Bad"),
			}
			assert.Panics(t, func() {
				decl.ObjType()
			})
		})
	})

	t.Run("Names", func(t *testing.T) {
		t.Run("model with swagger:model annotation uses Go name", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
				"User",
			)
			require.True(t, ok)

			name, goName := decl.Names()
			assert.EqualT(t, "User", goName)
			assert.EqualT(t, "User", name)
		})

		t.Run("type without model annotation returns Go name for both", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/operations",
				"SimpleOne",
			)
			require.True(t, ok)

			name, goName := decl.Names()
			assert.EqualT(t, "SimpleOne", goName)
			assert.EqualT(t, "SimpleOne", name)
		})

		t.Run("model with override name returns override", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
				"BaseStruct",
			)
			require.True(t, ok)

			name, goName := decl.Names()
			assert.EqualT(t, "BaseStruct", goName)
			assert.EqualT(t, "animal", name) // override name from swagger:model animal
		})
	})

	// Response name resolution moved to the grammar (grammar.ResponseBlock →
	// responses.Builder.ResponseName); the scanner no longer parses the
	// swagger:response argument. See the grammar parser tests and the
	// shared-parameters response coverage.

	// Operation-id targeting parse moved to the grammar
	// (grammar.ParametersBlock); see the grammar parser tests. The scanner
	// no longer parses swagger:parameters arguments.

	t.Run("HasAnnotation caching", func(t *testing.T) {
		t.Run("HasModelAnnotation caches result", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
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
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/operations",
				"GenericError",
			)
			require.True(t, ok)

			assert.True(t, decl.HasResponseAnnotation())
			// Second call: returns cached true
			assert.True(t, decl.HasResponseAnnotation())
		})

		t.Run("HasParameterAnnotation caches result", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/operations",
				"MyFileParams",
			)
			require.True(t, ok)

			assert.True(t, decl.HasParameterAnnotation())
			// Second call: returns cached true
			assert.True(t, decl.HasParameterAnnotation())
		})

		t.Run("HasModelAnnotation returns false for non-model", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/operations",
				"SimpleOne",
			)
			require.True(t, ok)

			assert.False(t, decl.HasModelAnnotation())
		})

		t.Run("HasResponseAnnotation returns false for non-response", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/operations",
				"SimpleOne",
			)
			require.True(t, ok)

			assert.False(t, decl.HasResponseAnnotation())
		})

		t.Run("HasParameterAnnotation returns false for non-parameter", func(t *testing.T) {
			decl, ok := sctx.FindDecl(
				"github.com/go-openapi/codescan/fixtures/goparsing/classification/operations",
				"SimpleOne",
			)
			require.True(t, ok)

			assert.False(t, decl.HasParameterAnnotation())
		})
	})
}
