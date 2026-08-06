// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	"golang.org/x/tools/go/packages"
)

// TestWrittenRHS_AgreesWithTheTypeChecker is the A/B behind WrittenRHS' scope resolution.
//
// The resolution exists to answer for a package the checker never saw the source of, where there is
// nothing to compare it against. So it is compared where there IS something: every type declaration
// in the fixture corpus, against the record go/types made for the same expression.
//
// Declarations the resolution declines (generic instantiations) are counted rather than compared —
// declining is the documented answer, and the count keeps an accidental widening visible.
func TestWrittenRHS_AgreesWithTheTypeChecker(t *testing.T) {
	var compared, declined int

	// Loaded root by root: the corpus deliberately holds trees that do not classify, and one of
	// those must not cost the walk every tree behind it.
	for _, root := range []string{
		"./goparsing/classification/...",
		"./goparsing/petstore/...",
		"./goparsing/spec/...",
		"./goparsing/go118/...",
		"./goparsing/go119/...",
		"./goparsing/go123/...",
		"./enhancements/...",
		"./bugs/...",
	} {
		sctx, err := NewScanCtx(&Options{Packages: []string{root}, WorkDir: "../../fixtures"})
		if err != nil {
			continue
		}

		for _, pkg := range sctx.app.AllPackages {
			if pkg.TypesInfo == nil || pkg.TypesInfo.Types == nil {
				continue
			}

			for _, decl := range typeSpecsOf(pkg) {
				recorded, known := pkg.TypesInfo.Types[decl.Spec.Type]
				if !known || recorded.Type == nil {
					continue
				}

				resolved, ok := decl.resolveTypeExpr(decl.Spec.Type)
				if !ok {
					declined++

					continue
				}
				compared++

				assert.True(t, types.Identical(resolved, recorded.Type),
					"%s.%s: resolved %s, the checker recorded %s",
					pkg.PkgPath, decl.Name(), resolved, recorded.Type,
				)
			}
		}
	}

	// The corpus is large enough that a resolution which quietly stopped resolving would show up
	// here rather than as a silent pass.
	assert.Greater(t, compared, 500)
	t.Logf("compared %d declarations, declined %d", compared, declined)
}

func TestWrittenRHS(t *testing.T) {
	sctx := loadClassificationPkgsCtx(t)
	const modelsPkg = "github.com/go-openapi/codescan/fixtures/goparsing/classification/models"

	t.Run("resolves without the checker's expression records", func(t *testing.T) {
		// The state a dependency served from compiled export data is in: complete types, no record
		// of what any expression denotes. WrittenRHS must still name the type that was written.
		for _, tc := range []struct {
			decl string
			want string
		}{
			{decl: "SomeStringType", want: "string"},                               // predeclared
			{decl: "SomeTimeType", want: "time.Time"},                              // qualified, imported
			{decl: "SomethingType", want: modelsPkg + ".Something"},                // package-level
			{decl: "SomeTimedType", want: "github.com/go-openapi/strfmt.DateTime"}, // qualified
			{decl: "SomeStringsType", want: "[]string"},                            // a type literal
			{decl: "SomeTimeMap", want: "map[string]time.Time"},                    // a type literal
			{decl: "SomeStringTypeAlias", want: "string"},                          // alias syntax
		} {
			t.Run(tc.decl, func(t *testing.T) {
				found, ok := sctx.FindDecl(modelsPkg, tc.decl)
				require.True(t, ok)

				stripped := *found
				stripped.Pkg = &packages.Package{PkgPath: found.Pkg.PkgPath} // no TypesInfo at all

				rhs, ok := stripped.WrittenRHS()
				require.True(t, ok)
				assert.EqualT(t, tc.want, rhs.String())
			})
		}
	})

	t.Run("declines a declaration with no syntax", func(t *testing.T) {
		found, ok := sctx.FindDecl(modelsPkg, "SomethingType")
		require.True(t, ok)

		decl := &EntityDecl{Type: found.Type, Alias: found.Alias}
		_, ok = decl.WrittenRHS()
		assert.False(t, ok, "without source there is no way to tell a named right-hand side from a literal")
	})
}

// typeSpecsOf returns one EntityDecl per top-level type declaration in pkg, annotated or not.
func typeSpecsOf(pkg *packages.Package) []*EntityDecl {
	var out []*EntityDecl

	for _, file := range pkg.Syntax {
		for _, d := range file.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, sp := range gd.Specs {
				ts, ok := sp.(*ast.TypeSpec)
				if !ok {
					continue
				}
				def, ok := pkg.TypesInfo.Defs[ts.Name]
				if !ok || def == nil {
					continue
				}
				nt, _ := def.Type().(*types.Named)
				at, _ := def.Type().(*types.Alias)
				if nt == nil && at == nil {
					continue
				}
				out = append(out, &EntityDecl{Type: nt, Alias: at, Ident: ts.Name, Spec: ts, File: file, Pkg: pkg})
			}
		}
	}

	return out
}
