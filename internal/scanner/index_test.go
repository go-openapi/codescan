// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"errors"
	"go/ast"
	"go/token"
	"slices"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	"golang.org/x/tools/go/packages"
)

func TestShouldAcceptTag(t *testing.T) {
	tagTests := []struct {
		tags        []string
		includeTags map[string]bool
		excludeTags map[string]bool
		expected    bool
	}{
		{nil, nil, nil, true},
		{[]string{"app"}, map[string]bool{"app": true}, nil, true},
		{[]string{"app"}, nil, map[string]bool{"app": true}, false},
	}
	for _, tt := range tagTests {
		actual := shouldAcceptTag(tt.tags, tt.includeTags, tt.excludeTags)
		assert.EqualT(t, tt.expected, actual)
	}
}

func TestShouldAcceptPkg(t *testing.T) {
	pkgTests := []struct {
		path        string
		includePkgs []string
		excludePkgs []string
		expected    bool
	}{
		{"", nil, nil, true},
		{"", nil, []string{"app"}, true},
		{"", []string{"app"}, nil, false},
		{"app", []string{"app"}, nil, true},
		{"app", nil, []string{"app"}, false},
		{"vendor/app", []string{"app"}, nil, true},
		{"vendor/app", nil, []string{"app"}, false},
	}
	for _, tt := range pkgTests {
		actual := shouldAcceptPkg(tt.path, tt.includePkgs, tt.excludePkgs)
		assert.EqualT(t, tt.expected, actual)
	}
}

func TestDetectNodes_UnknownAnnotation(t *testing.T) {
	file := &ast.File{
		Comments: []*ast.CommentGroup{
			{
				List: []*ast.Comment{
					{Text: "// swagger:bogusAnnotation"},
				},
			},
		},
	}

	idx := &TypeIndex{}
	_, err := idx.detectNodes(file)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrScanner))
}

func TestDetectNodes_AllAnnotationTypes(t *testing.T) {
	t.Run("meta node", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// swagger:meta"}}},
			},
		}
		idx := &TypeIndex{}
		n, err := idx.detectNodes(file)
		require.NoError(t, err)
		assert.True(t, n&metaNode != 0)
	})

	t.Run("route node", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// swagger:route GET /pets pets listPets"}}},
			},
		}
		idx := &TypeIndex{}
		n, err := idx.detectNodes(file)
		require.NoError(t, err)
		assert.True(t, n&routeNode != 0)
	})

	t.Run("operation node", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// swagger:operation GET /pets pets getPet"}}},
			},
		}
		idx := &TypeIndex{}
		n, err := idx.detectNodes(file)
		require.NoError(t, err)
		assert.True(t, n&operationNode != 0)
	})

	t.Run("model node", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// swagger:model"}}},
			},
		}
		idx := &TypeIndex{}
		n, err := idx.detectNodes(file)
		require.NoError(t, err)
		assert.True(t, n&modelNode != 0)
	})

	t.Run("parameters node", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// swagger:parameters myOp"}}},
			},
		}
		idx := &TypeIndex{}
		n, err := idx.detectNodes(file)
		require.NoError(t, err)
		assert.True(t, n&parametersNode != 0)
	})

	t.Run("response node", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// swagger:response myResp"}}},
			},
		}
		idx := &TypeIndex{}
		n, err := idx.detectNodes(file)
		require.NoError(t, err)
		assert.True(t, n&responseNode != 0)
	})

	t.Run("ignore annotation is accepted", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// swagger:ignore MyType"}}},
			},
		}
		idx := &TypeIndex{}
		_, err := idx.detectNodes(file)
		require.NoError(t, err)
	})

	t.Run("allOf annotation is accepted", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// swagger:allOf MyParent"}}},
			},
		}
		idx := &TypeIndex{}
		_, err := idx.detectNodes(file)
		require.NoError(t, err)
	})

	t.Run("known non-struct annotations are accepted", func(t *testing.T) {
		for _, annotation := range []string{"strfmt", "name", "discriminated", "file", "enum", "default", "alias", "type"} {
			file := &ast.File{
				Comments: []*ast.CommentGroup{
					{List: []*ast.Comment{{Text: "// swagger:" + annotation + " something"}}},
				},
			}
			idx := &TypeIndex{}
			_, err := idx.detectNodes(file)
			require.NoError(t, err, "annotation %q should be accepted", annotation)
		}
	})

	t.Run("no annotations at all", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// just a regular comment"}}},
			},
		}
		idx := &TypeIndex{}
		n, err := idx.detectNodes(file)
		require.NoError(t, err)
		assert.EqualT(t, node(0), n)
	})

	t.Run("nil comment line is skipped", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{nil, {Text: "// swagger:meta"}}},
			},
		}
		idx := &TypeIndex{}
		n, err := idx.detectNodes(file)
		require.NoError(t, err)
		assert.True(t, n&metaNode != 0)
	})
}

func TestCheckStructConflict(t *testing.T) {
	t.Run("no conflict with same annotation", func(t *testing.T) {
		seen := ""
		err := checkStructConflict(&seen, "model", "// swagger:model Foo")
		require.NoError(t, err)
		assert.EqualT(t, "model", seen)

		// Same annotation again: no conflict.
		err = checkStructConflict(&seen, "model", "// swagger:model Bar")
		require.NoError(t, err)
	})

	t.Run("conflict with different struct annotations", func(t *testing.T) {
		seen := "model"
		err := checkStructConflict(&seen, "parameters", "// swagger:parameters myOp")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner))
	})

	t.Run("model then response conflicts", func(t *testing.T) {
		seen := "model"
		err := checkStructConflict(&seen, "response", "// swagger:response myResp")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner))
	})

	t.Run("parameters then model conflicts", func(t *testing.T) {
		seen := "parameters"
		err := checkStructConflict(&seen, "model", "// swagger:model Foo")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner))
	})
}

func TestDetectNodes_StructConflict(t *testing.T) {
	t.Run("model and parameters in same comment group", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{
					List: []*ast.Comment{
						{Text: "// swagger:model Foo"},
						{Text: "// swagger:parameters myOp"},
					},
				},
			},
		}
		idx := &TypeIndex{}
		_, err := idx.detectNodes(file)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner))
	})

	t.Run("model and response in same comment group", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{
					List: []*ast.Comment{
						{Text: "// swagger:model Foo"},
						{Text: "// swagger:response myResp"},
					},
				},
			},
		}
		idx := &TypeIndex{}
		_, err := idx.detectNodes(file)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner))
	})

	t.Run("parameters and response in same comment group", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{
					List: []*ast.Comment{
						{Text: "// swagger:parameters myOp"},
						{Text: "// swagger:response myResp"},
					},
				},
			},
		}
		idx := &TypeIndex{}
		_, err := idx.detectNodes(file)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner))
	})

	t.Run("model in separate comment groups is fine", func(t *testing.T) {
		file := &ast.File{
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// swagger:model Foo"}}},
				{List: []*ast.Comment{{Text: "// swagger:response myResp"}}},
			},
		}
		idx := &TypeIndex{}
		n, err := idx.detectNodes(file)
		require.NoError(t, err)
		assert.True(t, n&modelNode != 0)
		assert.True(t, n&responseNode != 0)
	})
}

func TestNewTypeIndex_ExcludeDeps(t *testing.T) {
	sctx, err := NewScanCtx(&Options{
		Packages: []string{
			"./goparsing/classification",
			"./goparsing/classification/models",
			"./goparsing/classification/operations",
		},
		WorkDir:     "../../fixtures",
		ExcludeDeps: true,
	})
	require.NoError(t, err)
	require.NotNil(t, sctx)
	require.NotNil(t, sctx.app)

	// With ExcludeDeps, imports should NOT be walked:
	// the AllPackages map should only contain the explicitly-loaded packages,
	// NOT transitive dependencies like strfmt.
	_, hasStrfmt := sctx.app.AllPackages["github.com/go-openapi/strfmt"]
	assert.False(t, hasStrfmt, "strfmt should not be indexed when ExcludeDeps is true")
}

func TestNewTypeIndex_IncludeExcludePkgs(t *testing.T) {
	t.Run("include only models package", func(t *testing.T) {
		sctx, err := NewScanCtx(&Options{
			Packages: []string{
				"./goparsing/classification",
				"./goparsing/classification/models",
				"./goparsing/classification/operations",
			},
			WorkDir:     "../../fixtures",
			ExcludeDeps: true,
			Include:     []string{"models"},
		})
		require.NoError(t, err)

		// Only the models package should have been processed for swagger annotations.
		// Routes and parameters come from the operations package, so they should be empty.
		assert.Empty(t, sctx.app.Routes)
		assert.Empty(t, sctx.app.Parameters)
	})

	t.Run("exclude operations package", func(t *testing.T) {
		sctx, err := NewScanCtx(&Options{
			Packages: []string{
				"./goparsing/classification",
				"./goparsing/classification/models",
				"./goparsing/classification/operations",
			},
			WorkDir:     "../../fixtures",
			ExcludeDeps: true,
			Exclude:     []string{"operations$"},
		})
		require.NoError(t, err)

		// Routes and parameters come from operations, so excluding it should eliminate them.
		assert.Empty(t, sctx.app.Routes)
		assert.Empty(t, sctx.app.Parameters)
	})
}

func TestNewTypeIndex_IncludeExcludeTags(t *testing.T) {
	t.Run("include tag filters routes", func(t *testing.T) {
		sctx, err := NewScanCtx(&Options{
			Packages: []string{
				"./goparsing/classification",
				"./goparsing/classification/operations",
			},
			WorkDir:     "../../fixtures",
			ExcludeDeps: true,
			IncludeTags: []string{"orders"},
		})
		require.NoError(t, err)

		// Only routes tagged "orders" should be included.
		for r := range sctx.Routes() {
			assert.True(t, slices.Contains(r.Tags, "orders"), "expected route to have tag 'orders', got tags: %v", r.Tags)
		}
	})

	t.Run("exclude tag filters routes", func(t *testing.T) {
		sctx, err := NewScanCtx(&Options{
			Packages: []string{
				"./goparsing/classification",
				"./goparsing/classification/operations",
			},
			WorkDir:     "../../fixtures",
			ExcludeDeps: true,
			ExcludeTags: []string{"orders"},
		})
		require.NoError(t, err)

		// No route should have the "orders" tag.
		for r := range sctx.Routes() {
			for _, tag := range r.Tags {
				assert.True(t, tag != "orders", "route should not have tag 'orders', got tags: %v", r.Tags)
			}
		}
	})
}

func TestCollectOperationPathAnnotations(t *testing.T) {
	sctx, err := NewScanCtx(&Options{
		Packages: []string{"./goparsing/classification/operations_annotation"},
		WorkDir:  "../../fixtures",
	})
	require.NoError(t, err)

	// operations_annotation has swagger:operation annotations.
	var count int
	for range sctx.Operations() {
		count++
	}
	assert.True(t, count > 0, "expected at least one operation from operations_annotation fixture")
}

func TestCollectOperationPathAnnotations_TagFiltering(t *testing.T) {
	t.Run("include tag filters operations", func(t *testing.T) {
		sctx, err := NewScanCtx(&Options{
			Packages:    []string{"./goparsing/classification/operations_annotation"},
			WorkDir:     "../../fixtures",
			ExcludeDeps: true,
			IncludeTags: []string{"Events"},
		})
		require.NoError(t, err)

		var count int
		for op := range sctx.Operations() {
			count++
			// All remaining operations should have the "Events" tag.
			assert.True(t, slices.Contains(op.Tags, "Events"), "expected operation to have tag 'Events', got tags: %v", op.Tags)
		}
		// Should have filtered some operations out.
		assert.True(t, count > 0, "expected at least one operation with tag 'Events'")
	})

	t.Run("exclude tag filters operations", func(t *testing.T) {
		sctx, err := NewScanCtx(&Options{
			Packages:    []string{"./goparsing/classification/operations_annotation"},
			WorkDir:     "../../fixtures",
			ExcludeDeps: true,
			ExcludeTags: []string{"pets"},
		})
		require.NoError(t, err)

		for op := range sctx.Operations() {
			for _, tag := range op.Tags {
				assert.True(t, tag != "pets", "operation should not have tag 'pets', got tags: %v", op.Tags)
			}
		}
	})
}

func TestNewTypeIndex_ErrorPropagation(t *testing.T) {
	t.Run("struct annotation conflict propagates error through build chain", func(t *testing.T) {
		// The invalid_model_param fixture has swagger:model and swagger:parameters
		// on the same struct, which triggers a struct conflict error in detectNodes,
		// propagated through processFile → processPackage → build → NewTypeIndex → NewScanCtx.
		_, err := NewScanCtx(&Options{
			Packages: []string{"./goparsing/invalid_model_param"},
			WorkDir:  "../../fixtures",
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner), "expected ErrScanner, got: %v", err)
	})

	t.Run("model and response conflict propagates error", func(t *testing.T) {
		_, err := NewScanCtx(&Options{
			Packages: []string{"./goparsing/invalid_model_response"},
			WorkDir:  "../../fixtures",
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner), "expected ErrScanner, got: %v", err)
	})

	t.Run("param and model conflict propagates error", func(t *testing.T) {
		_, err := NewScanCtx(&Options{
			Packages: []string{"./goparsing/invalid_param_model"},
			WorkDir:  "../../fixtures",
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner), "expected ErrScanner, got: %v", err)
	})

	t.Run("response and model conflict propagates error", func(t *testing.T) {
		_, err := NewScanCtx(&Options{
			Packages: []string{"./goparsing/invalid_response_model"},
			WorkDir:  "../../fixtures",
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner), "expected ErrScanner, got: %v", err)
	})

	t.Run("duplicate package is de-duplicated", func(t *testing.T) {
		// Load the same package twice — the second occurrence should be
		// skipped (line 109 in build).
		sctx, err := NewScanCtx(&Options{
			Packages: []string{
				"./goparsing/petstore/enums",
				"./goparsing/petstore/enums", // duplicate
			},
			WorkDir: "../../fixtures",
		})
		require.NoError(t, err)
		require.NotNil(t, sctx)
	})

	t.Run("walkImports error propagation", func(t *testing.T) {
		// Load a valid package alongside an invalid one that shares imports.
		// The invalid_model_param package is a main package; loading it should
		// trigger the conflict error even if other packages are fine.
		_, err := NewScanCtx(&Options{
			Packages: []string{
				"./goparsing/petstore/enums",
				"./goparsing/invalid_model_param",
			},
			WorkDir: "../../fixtures",
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrScanner))
	})
}

func TestProcessFileDecls_BadDecl(t *testing.T) {
	// Ensure processFileDecls skips BadDecl without panicking.
	idx := &TypeIndex{
		AllPackages: make(map[string]*packages.Package),
		Models:      make(map[*ast.Ident]*EntityDecl),
		ExtraModels: make(map[*ast.Ident]*EntityDecl),
	}

	file := &ast.File{
		Decls: []ast.Decl{
			&ast.BadDecl{},
			&ast.GenDecl{
				Tok:   token.IMPORT,
				Specs: []ast.Spec{&ast.ImportSpec{Path: &ast.BasicLit{Value: `"fmt"`}}},
			},
		},
	}

	// Should not panic.
	assert.NotPanics(t, func() {
		idx.processFileDecls(nil, file, 0)
	})
}
