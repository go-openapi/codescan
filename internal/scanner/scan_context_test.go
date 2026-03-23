// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

var classificationCtx *ScanCtx //nolint:gochecknoglobals // test package cache shared across test functions

func TestApplication_LoadCode(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)
	require.NotNil(t, sctx)
	require.NotNil(t, sctx.app)
	require.Len(t, sctx.app.Models, 39)
	require.Len(t, sctx.app.Meta, 1)
	require.Len(t, sctx.app.Routes, 7)
	require.Empty(t, sctx.app.Operations)
	require.Len(t, sctx.app.Parameters, 10)
	require.Len(t, sctx.app.Responses, 11)
}

func TestScanCtx_OptionAccessors(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	// Default options: all false.
	assert.False(t, sctx.SkipExtensions())
	assert.False(t, sctx.DescWithRef())
	assert.False(t, sctx.SetXNullableForPointers())
	assert.False(t, sctx.TransparentAliases())
	assert.False(t, sctx.RefAliases())
	assert.False(t, sctx.Debug())
}

func TestScanCtx_OptionAccessors_Enabled(t *testing.T) {
	sctx, err := NewScanCtx(&Options{
		Packages:                []string{"./goparsing/classification"},
		WorkDir:                 "../../fixtures",
		SkipExtensions:          true,
		DescWithRef:             true,
		SetXNullableForPointers: true,
		TransparentAliases:      true,
		RefAliases:              true,
		Debug:                   true,
	})
	require.NoError(t, err)

	assert.True(t, sctx.SkipExtensions())
	assert.True(t, sctx.DescWithRef())
	assert.True(t, sctx.SetXNullableForPointers())
	assert.True(t, sctx.TransparentAliases())
	assert.True(t, sctx.RefAliases())
	assert.True(t, sctx.Debug())
}

func TestScanCtx_Meta(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	var count int
	for range sctx.Meta() {
		count++
	}
	assert.EqualT(t, 1, count)
}

func TestScanCtx_Operations(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	var count int
	for range sctx.Operations() {
		count++
	}
	// The classification fixture has no operation annotations (only route annotations).
	assert.EqualT(t, 0, count)
}

func TestScanCtx_Routes(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	var count int
	for range sctx.Routes() {
		count++
	}
	assert.EqualT(t, 7, count)
}

func TestScanCtx_Responses(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	var count int
	for range sctx.Responses() {
		count++
	}
	assert.EqualT(t, 11, count)
}

func TestScanCtx_Parameters(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	var count int
	for range sctx.Parameters() {
		count++
	}
	assert.EqualT(t, 10, count)
}

func TestScanCtx_Models(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	var count int
	for range sctx.Models() {
		count++
	}
	assert.EqualT(t, 39, count)
}

func TestScanCtx_ExtraModels(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	// No extra models initially.
	assert.EqualT(t, 0, sctx.NumExtraModels())

	var count int
	for range sctx.ExtraModels() {
		count++
	}
	assert.EqualT(t, 0, count)
}

func TestScanCtx_NilApp(t *testing.T) {
	// Test all iterator/accessor methods when app is nil.
	sctx := &ScanCtx{}

	assert.Nil(t, sctx.Meta())
	assert.Nil(t, sctx.Operations())
	assert.Nil(t, sctx.Routes())
	assert.Nil(t, sctx.Responses())
	assert.Nil(t, sctx.Parameters())
	assert.Nil(t, sctx.Models())
	assert.Nil(t, sctx.ExtraModels())
	assert.EqualT(t, 0, sctx.NumExtraModels())
}

func TestScanCtx_PkgForPath(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	t.Run("known package", func(t *testing.T) {
		pkg, ok := sctx.PkgForPath("github.com/go-openapi/codescan/fixtures/goparsing/classification/models")
		assert.True(t, ok)
		assert.NotNil(t, pkg)
		assert.EqualT(t, "models", pkg.Name)
	})

	t.Run("unknown package", func(t *testing.T) {
		_, ok := sctx.PkgForPath("github.com/nonexistent/package")
		assert.False(t, ok)
	})
}

func TestScanCtx_FindDecl(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	t.Run("finds existing type", func(t *testing.T) {
		decl, ok := sctx.FindDecl(
			"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
			"User",
		)
		require.True(t, ok)
		assert.NotNil(t, decl)
		assert.EqualT(t, "User", decl.Ident.Name)
		assert.NotNil(t, decl.Pkg)
		assert.NotNil(t, decl.File)
		assert.NotNil(t, decl.Spec)
	})

	t.Run("unknown package returns false", func(t *testing.T) {
		_, ok := sctx.FindDecl("github.com/nonexistent/package", "Foo")
		assert.False(t, ok)
	})

	t.Run("unknown type in known package returns false", func(t *testing.T) {
		_, ok := sctx.FindDecl(
			"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
			"NonExistentType",
		)
		assert.False(t, ok)
	})
}

func TestScanCtx_FindModel(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	t.Run("finds model in Models map", func(t *testing.T) {
		decl, ok := sctx.FindModel(
			"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
			"User",
		)
		require.True(t, ok)
		assert.NotNil(t, decl)
		assert.EqualT(t, "User", decl.Ident.Name)
	})

	t.Run("finds type not in Models, adds to ExtraModels", func(t *testing.T) {
		// SimpleOne has no swagger:model annotation, but it exists as a type.
		beforeExtra := sctx.NumExtraModels()

		decl, ok := sctx.FindModel(
			"github.com/go-openapi/codescan/fixtures/goparsing/classification/operations",
			"SimpleOne",
		)
		require.True(t, ok)
		assert.NotNil(t, decl)
		assert.EqualT(t, "SimpleOne", decl.Ident.Name)

		// Should have been added to ExtraModels.
		assert.True(t, sctx.NumExtraModels() > beforeExtra)
	})

	t.Run("type that does not exist returns false", func(t *testing.T) {
		_, ok := sctx.FindModel(
			"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
			"NonExistentType",
		)
		assert.False(t, ok)
	})
}

func TestScanCtx_MoveExtraToModel(t *testing.T) {
	sctx, err := NewScanCtx(&Options{
		Packages: []string{
			"./goparsing/classification",
			"./goparsing/classification/models",
			"./goparsing/classification/operations",
		},
		WorkDir: "../../fixtures",
	})
	require.NoError(t, err)

	// Find a type that will be added to ExtraModels.
	decl, ok := sctx.FindModel(
		"github.com/go-openapi/codescan/fixtures/goparsing/classification/operations",
		"SimpleOne",
	)
	require.True(t, ok)

	numModels := len(sctx.app.Models)
	numExtras := sctx.NumExtraModels()
	require.True(t, numExtras > 0, "expected at least one extra model")

	// Move it to models.
	sctx.MoveExtraToModel(decl.Ident)

	assert.EqualT(t, numModels+1, len(sctx.app.Models))
	assert.EqualT(t, numExtras-1, sctx.NumExtraModels())

	// Moving a non-existent key should be a no-op.
	sctx.MoveExtraToModel(ast.NewIdent("nonexistent"))
	assert.EqualT(t, numModels+1, len(sctx.app.Models))
}

func TestScanCtx_DeclForType(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	t.Run("named type", func(t *testing.T) {
		decl, ok := sctx.FindDecl(
			"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
			"User",
		)
		require.True(t, ok)
		require.NotNil(t, decl.Type)

		found, ok := sctx.DeclForType(decl.Type)
		assert.True(t, ok)
		assert.EqualT(t, "User", found.Ident.Name)
	})

	t.Run("pointer to named type", func(t *testing.T) {
		decl, ok := sctx.FindDecl(
			"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
			"User",
		)
		require.True(t, ok)
		require.NotNil(t, decl.Type)

		// Wrap in a pointer type to test the *types.Pointer branch.
		ptrType := types.NewPointer(decl.Type)
		found, ok := sctx.DeclForType(ptrType)
		assert.True(t, ok)
		assert.EqualT(t, "User", found.Ident.Name)
	})

	t.Run("basic type returns false", func(t *testing.T) {
		// types.Typ[types.Int] is a *types.Basic
		_, ok := sctx.DeclForType(types.Typ[types.Int])
		assert.False(t, ok)
	})
}

func TestScanCtx_DeclForType_Alias(t *testing.T) {
	// Load the spec fixture which has type aliases (Customer = User).
	sctx, err := NewScanCtx(&Options{
		Packages: []string{"./goparsing/spec"},
		WorkDir:  "../../fixtures",
	})
	require.NoError(t, err)

	decl, ok := sctx.FindDecl(
		"github.com/go-openapi/codescan/fixtures/goparsing/spec",
		"Customer",
	)
	require.True(t, ok)
	require.NotNil(t, decl.Alias, "Customer should be a type alias")

	found, ok := sctx.DeclForType(decl.Alias)
	assert.True(t, ok)
	assert.EqualT(t, "Customer", found.Ident.Name)
}

func TestScanCtx_PkgForType(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	t.Run("named type returns package", func(t *testing.T) {
		decl, ok := sctx.FindDecl(
			"github.com/go-openapi/codescan/fixtures/goparsing/classification/models",
			"User",
		)
		require.True(t, ok)
		require.NotNil(t, decl.Type)

		pkg, ok := sctx.PkgForType(decl.Type)
		assert.True(t, ok)
		assert.EqualT(t, "models", pkg.Name)
	})

	t.Run("basic type returns false", func(t *testing.T) {
		_, ok := sctx.PkgForType(types.Typ[types.String])
		assert.False(t, ok)
	})
}

func TestScanCtx_PkgForType_Alias(t *testing.T) {
	// Load the spec fixture which has type aliases (Customer = User).
	sctx, err := NewScanCtx(&Options{
		Packages: []string{"./goparsing/spec"},
		WorkDir:  "../../fixtures",
	})
	require.NoError(t, err)

	decl, ok := sctx.FindDecl(
		"github.com/go-openapi/codescan/fixtures/goparsing/spec",
		"Customer",
	)
	require.True(t, ok)
	require.NotNil(t, decl.Alias, "Customer should be a type alias")

	pkg, ok := sctx.PkgForType(decl.Alias)
	assert.True(t, ok)
	assert.EqualT(t, "spec", pkg.Name)
}

func TestScanCtx_FindComments(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)
	pkg, ok := sctx.PkgForPath("github.com/go-openapi/codescan/fixtures/goparsing/classification/models")
	require.True(t, ok)

	t.Run("finds comments for existing type", func(t *testing.T) {
		comments, ok := sctx.FindComments(pkg, "User")
		assert.True(t, ok)
		assert.NotNil(t, comments)
	})

	t.Run("returns false for unknown type", func(t *testing.T) {
		_, ok := sctx.FindComments(pkg, "NonExistent")
		assert.False(t, ok)
	})
}

func TestScanCtx_FindEnumValues(t *testing.T) {
	// Load the petstore fixture which has enum types.
	sctx, err := NewScanCtx(&Options{
		Packages: []string{"./goparsing/petstore/enums"},
		WorkDir:  "../../fixtures",
	})
	require.NoError(t, err)

	pkg, ok := sctx.PkgForPath("github.com/go-openapi/codescan/fixtures/goparsing/petstore/enums")
	require.True(t, ok)

	t.Run("finds enum values for Status type", func(t *testing.T) {
		list, descList, ok := sctx.FindEnumValues(pkg, "Status")
		assert.True(t, ok)
		require.Len(t, list, 3) // available, pending, sold
		require.Len(t, descList, 3)

		// Verify the string values were extracted.
		assert.EqualT(t, "available", list[0])
		assert.EqualT(t, "pending", list[1])
		assert.EqualT(t, "sold", list[2])

		// Descriptions should contain the literal value and the constant name.
		for _, desc := range descList {
			assert.True(t, desc != "", "description should not be empty")
		}
	})

	t.Run("finds enum values for Priority type with doc comments", func(t *testing.T) {
		list, descList, ok := sctx.FindEnumValues(pkg, "Priority")
		assert.True(t, ok)
		require.Len(t, list, 2) // PriorityLow=1, PriorityHigh=2
		require.Len(t, descList, 2)

		// Verify int values were extracted.
		v0, ok0 := list[0].(int64)
		require.True(t, ok0, "expected int64 value, got %T", list[0])
		assert.EqualT(t, int64(1), v0)

		v1, ok1 := list[1].(int64)
		require.True(t, ok1, "expected int64 value, got %T", list[1])
		assert.EqualT(t, int64(2), v1)

		// Description for PriorityLow should contain the doc comment text.
		// It has a multi-line doc comment ("// low priority\n// items are handled last").
		assert.True(t, len(descList[0]) > 0, "description should contain doc comment text")
	})

	t.Run("non-matching enum name returns empty", func(t *testing.T) {
		list, descList, ok := sctx.FindEnumValues(pkg, "NonExistentEnum")
		assert.True(t, ok) // always returns true
		assert.Empty(t, list)
		assert.Empty(t, descList)
	})
}

func TestScanCtx_FindEnumValues_NoConsts(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)

	// Use a package that has types but no related const enums.
	pkg, ok := sctx.PkgForPath("github.com/go-openapi/codescan/fixtures/goparsing/classification/models")
	require.True(t, ok)

	list, descList, ok := sctx.FindEnumValues(pkg, "User")
	assert.True(t, ok)
	assert.Empty(t, list)
	assert.Empty(t, descList)
}

func TestNewScanCtx_WithBuildTags(t *testing.T) {
	sctx, err := NewScanCtx(&Options{
		Packages:  []string{"./goparsing/classification"},
		WorkDir:   "../../fixtures",
		BuildTags: "integration",
	})
	require.NoError(t, err)
	require.NotNil(t, sctx)
}

func TestNewScanCtx_InvalidPackage(t *testing.T) {
	// Loading a nonexistent package should succeed at the Load level
	// but produce no useful data (packages.Load doesn't return errors for missing pkgs,
	// it returns packages with errors embedded).
	sctx, err := NewScanCtx(&Options{
		Packages: []string{"./nonexistent"},
		WorkDir:  "../../fixtures",
	})
	// This may or may not error depending on go/packages behavior;
	// what matters is it doesn't panic.
	if err != nil {
		return
	}
	require.NotNil(t, sctx)
}

func TestScanCtx_findEnumValue_EdgeCases(t *testing.T) {
	sctx := &ScanCtx{}

	t.Run("non-ValueSpec returns nil", func(t *testing.T) {
		spec := &ast.ImportSpec{Path: &ast.BasicLit{Value: `"fmt"`}}
		val, desc := sctx.findEnumValue(spec, "Foo")
		assert.Nil(t, val)
		assert.EqualT(t, "", desc)
	})

	t.Run("ValueSpec with nil Type returns nil", func(t *testing.T) {
		spec := &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("X")},
		}
		val, desc := sctx.findEnumValue(spec, "Foo")
		assert.Nil(t, val)
		assert.EqualT(t, "", desc)
	})

	t.Run("ValueSpec with selector type returns nil", func(t *testing.T) {
		// Type is *ast.SelectorExpr, not *ast.Ident
		spec := &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("X")},
			Type:  &ast.SelectorExpr{X: ast.NewIdent("pkg"), Sel: ast.NewIdent("Type")},
		}
		val, desc := sctx.findEnumValue(spec, "Foo")
		assert.Nil(t, val)
		assert.EqualT(t, "", desc)
	})

	t.Run("ValueSpec with non-matching enum name returns nil", func(t *testing.T) {
		spec := &ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent("X")},
			Type:   ast.NewIdent("Bar"),
			Values: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "1"}},
		}
		val, desc := sctx.findEnumValue(spec, "Foo")
		assert.Nil(t, val)
		assert.EqualT(t, "", desc)
	})

	t.Run("ValueSpec with no values returns nil", func(t *testing.T) {
		spec := &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("X")},
			Type:  ast.NewIdent("Foo"),
		}
		val, desc := sctx.findEnumValue(spec, "Foo")
		assert.Nil(t, val)
		assert.EqualT(t, "", desc)
	})

	t.Run("ValueSpec with non-BasicLit value returns nil", func(t *testing.T) {
		spec := &ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent("X")},
			Type:   ast.NewIdent("Foo"),
			Values: []ast.Expr{ast.NewIdent("someFunc")}, // *ast.Ident, not *ast.BasicLit
		}
		val, desc := sctx.findEnumValue(spec, "Foo")
		assert.Nil(t, val)
		assert.EqualT(t, "", desc)
	})

	t.Run("ValueSpec with multiple names builds description", func(t *testing.T) {
		spec := &ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent("A"), ast.NewIdent("B")},
			Type:   ast.NewIdent("Foo"),
			Values: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "42"}},
		}
		val, desc := sctx.findEnumValue(spec, "Foo")
		require.NotNil(t, val)
		intVal, ok := val.(int64)
		require.True(t, ok)
		assert.EqualT(t, int64(42), intVal)
		// Description should contain "42 A B"
		assert.True(t, len(desc) > 0)
	})

	t.Run("ValueSpec with doc comments builds description", func(t *testing.T) {
		spec := &ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent("X")},
			Type:   ast.NewIdent("Foo"),
			Values: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"hello"`}},
			Doc: &ast.CommentGroup{
				List: []*ast.Comment{
					{Text: "// first line"},
					{Text: "// second line"},
				},
			},
		}
		val, desc := sctx.findEnumValue(spec, "Foo")
		require.NotNil(t, val)
		assert.EqualT(t, "hello", val)
		// Description should contain the doc comment text.
		assert.True(t, len(desc) > 0)
	})
}

func TestSliceToSet(t *testing.T) {
	t.Run("empty slice produces empty map", func(t *testing.T) {
		result := sliceToSet(nil)
		assert.EqualT(t, 0, len(result))
	})

	t.Run("populates map correctly", func(t *testing.T) {
		result := sliceToSet([]string{"a", "b", "c"})
		assert.EqualT(t, 3, len(result))
		assert.True(t, result["a"])
		assert.True(t, result["b"])
		assert.True(t, result["c"])
	})

	t.Run("deduplicates entries", func(t *testing.T) {
		result := sliceToSet([]string{"x", "x", "y"})
		assert.EqualT(t, 2, len(result))
	})
}

func loadClassificationPkgsCtx(t *testing.T) *ScanCtx {
	t.Helper()

	if classificationCtx != nil {
		return classificationCtx
	}

	sctx, err := NewScanCtx(&Options{
		Packages: []string{
			"./goparsing/classification",
			"./goparsing/classification/models",
			"./goparsing/classification/operations",
		},
		WorkDir: "../../fixtures",
	})
	require.NoError(t, err)
	classificationCtx = sctx

	return classificationCtx
}
