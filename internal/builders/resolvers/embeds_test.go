// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package resolvers_test

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/go-openapi/codescan/internal/builders/resolvers"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	"golang.org/x/tools/go/packages"
)

// TestEmbeds_AgreesWithTheTypeChecker is the A/B behind reading an embed's type from the declared
// type's underlying instead of from types.Info.
//
// Positional pairing between an AST member list and a *types.Struct / *types.Interface is where a
// bug would live, and it is not visible from the emitted spec — a mispaired embed produces a
// plausible wrong answer rather than a failure. So it is compared entry by entry, over every struct
// and interface declaration in the fixture corpus, against the record go/types made for the same
// expression.
func TestEmbeds_AgreesWithTheTypeChecker(t *testing.T) {
	var (
		declarations, paired, multiName int
		unreached                       []string
	)

	for _, pkg := range loadCorpus(t) {
		for _, decl := range typeSpecs(pkg) {
			name := decl.name
			members, ok := memberList(decl.spec)
			if !ok {
				continue
			}
			declarations++

			// What the type-checker says each anonymous member denotes.
			want := map[*ast.Field]types.Type{}
			for _, afld := range members {
				if len(afld.Names) > 1 {
					multiName++
				}
				if len(afld.Names) != 0 {
					continue
				}
				if tv, known := pkg.TypesInfo.Types[afld.Type]; known && tv.Type != nil {
					want[afld] = tv.Type
				}
			}

			got := map[*ast.Field]types.Type{}
			for _, embed := range resolvers.Embeds(members, decl.def.Type().Underlying()) {
				got[embed.Field] = embed.Type
			}

			for afld, wantType := range want {
				gotType, found := got[afld]
				if !found {
					unreached = append(unreached, pkg.PkgPath+"."+name)

					continue
				}
				paired++
				assert.True(t, types.Identical(gotType, wantType),
					"%s.%s: paired %s, the checker recorded %s", pkg.PkgPath, name, gotType, wantType,
				)
			}

			assert.Len(t, got, len(want),
				"%s.%s: paired %d anonymous members, the checker recorded %d", pkg.PkgPath, name, len(got), len(want),
			)
		}
	}

	assert.Empty(t, unreached, "anonymous members the pairing did not reach")
	// The corpus is large enough that a pairing which quietly stopped pairing would show up here
	// rather than as a silent pass — and a multi-name field (`A, B int`, one AST entry, two struct
	// fields) is what makes the positional walk non-trivial.
	assert.Greater(t, declarations, 500)
	assert.Positive(t, multiName)
	t.Logf("walked %d struct/interface declarations, paired %d embeds", declarations, paired)
}

func TestEmbeds_YieldsNothingWithoutMembers(t *testing.T) {
	assert.Empty(t, resolvers.Embeds(nil, types.Typ[types.String]))
	assert.Empty(t, resolvers.Embeds(nil, types.NewSlice(types.Typ[types.Int])))
	assert.Empty(t, resolvers.Embeds(nil, types.NewStruct(nil, nil)))
}

func loadCorpus(t *testing.T) []*packages.Package {
	t.Helper()

	loaded, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps,
		Dir:  scantest.FixturesDir(),
	}, "./goparsing/...", "./enhancements/...", "./bugs/...")
	require.NoError(t, err)

	// Type errors are expected: the corpus deliberately holds trees that do not build. Every
	// declaration that DID check still carries a usable record, which is what this walk reads.
	return loaded
}

type typeDecl struct {
	name string
	spec *ast.TypeSpec
	def  types.Object
}

// typeSpecs returns the top-level type declarations of pkg the checker resolved.
func typeSpecs(pkg *packages.Package) []typeDecl {
	if pkg.TypesInfo == nil || pkg.TypesInfo.Types == nil {
		return nil
	}

	var out []typeDecl
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
				if !ok || def == nil || def.Type() == nil {
					continue
				}
				out = append(out, typeDecl{name: ts.Name.Name, spec: ts, def: def})
			}
		}
	}

	return out
}

// memberList returns a declaration's struct fields or interface methods.
func memberList(spec *ast.TypeSpec) ([]*ast.Field, bool) {
	switch tpe := spec.Type.(type) {
	case *ast.StructType:
		if tpe.Fields == nil {
			return nil, false
		}

		return tpe.Fields.List, true
	case *ast.InterfaceType:
		if tpe.Methods == nil {
			return nil, false
		}

		return tpe.Methods.List, true
	default:
		return nil, false
	}
}
