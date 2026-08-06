// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package packages_test

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
	"testing/fstest"

	"golang.org/x/tools/go/gcexportdata"

	"github.com/go-openapi/codescan/internal/packages"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// exportFor type-checks src and writes it out the way the compiler would, so these tests need no toolchain and no
// GOROOT.
func exportFor(t *testing.T, pkgPath, src string) []byte {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, pkgPath+".go", src, parser.ParseComments)
	require.NoError(t, err)

	pkg, err := (&types.Config{}).Check(pkgPath, fset, []*ast.File{f}, nil)
	require.NoError(t, err)

	var blob bytes.Buffer
	require.NoError(t, gcexportdata.Write(&blob, fset, pkg))

	return blob.Bytes()
}

// Export data answers what the types are.
//
// It cannot answer what the package SAYS about them, because there are no comments in it — and for codescan that
// second answer is load-bearing: strfmt marks its own types, and those marks are what give a field its format.
//
// The two halves cannot be combined after the fact: go/types records what a type expression denotes behind an
// unexported field, so a package assembled from export data plus parsed syntax has declarations the builders read and
// no record of what they denote.
//
// So the choice is per package and whole — a dependency that says something is read from source, and the export data
// behind it still serves everything it in turn imports, which is where the saving was all along.
func TestExportData_KeepsDependencyComments(t *testing.T) {
	t.Parallel()

	const src = "package fakefmt\n\n// Stamp is a moment in time.\n//\n// swagger:strfmt date-time\ntype Stamp struct{ Seconds int64 }\n"

	tree := fstest.MapFS{
		"go.mod":          &fstest.MapFile{Data: []byte("module example.com/m\n\ngo 1.25.0\n\nrequire fakefmt v1.0.0\n\nreplace fakefmt => ./vendored\n")},
		"vendored/go.mod": &fstest.MapFile{Data: []byte("module fakefmt\n\ngo 1.25.0\n")},
		"vendored/f.go":   &fstest.MapFile{Data: []byte(src)},
		"p/p.go":          &fstest.MapFile{Data: []byte("package p\n\nimport \"fakefmt\"\n\ntype T struct{ At fakefmt.Stamp }\n")},
	}
	exports := fstest.MapFS{"fakefmt.export": &fstest.MapFile{Data: exportFor(t, "fakefmt", src)}}

	var exportOnly []packages.ExportOnly
	pkgs, err := packages.NewLoader(
		packages.WithFS(tree),
		packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}),
		packages.WithExportData(exports),
		packages.WithOnExportOnly(func(e packages.ExportOnly) { exportOnly = append(exportOnly, e) }),
	).Load(&packages.Config{}, "./p")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	dep := pkgs[0].Imports["fakefmt"]
	require.NotNil(t, dep, "a dependency served from export data is still a package")

	// The types came from the compiler.
	require.NotNil(t, dep.Types)
	assert.NotNil(t, dep.Types.Scope().Lookup("Stamp"))

	// It carries an annotation, so it was read from source and nothing about it is second-hand.
	require.NotEmpty(t, dep.Syntax, "a dependency that says something is read from its source")
	assert.Contains(t, commentsOf(dep.Syntax), "swagger:strfmt date-time")

	require.NotNil(t, dep.TypesInfo)
	assert.NotEmpty(t, dep.TypesInfo.Defs)
	assert.NotEmpty(t, dep.TypesInfo.Types,
		"including what each declaration denotes, which the builders assert on and export data cannot supply")

	assert.Empty(t, exportOnly, "nothing was lost, so nothing is announced")
}

// When only the types are available the scan still works, and says so.
func TestExportData_AnnouncesMissingSource(t *testing.T) {
	t.Parallel()

	const src = "package fakefmt\n\n// swagger:strfmt date-time\ntype Stamp struct{ Seconds int64 }\n"

	// The tree has no copy of fakefmt at all: only its export data.
	tree := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/m\n\ngo 1.25.0\n")},
		"p/p.go": &fstest.MapFile{Data: []byte("package p\n\nimport \"fakefmt\"\n\ntype T struct{ At fakefmt.Stamp }\n")},
	}
	exports := fstest.MapFS{"fakefmt.export": &fstest.MapFile{Data: exportFor(t, "fakefmt", src)}}

	var exportOnly []packages.ExportOnly
	pkgs, err := packages.NewLoader(
		packages.WithFS(tree),
		packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}),
		packages.WithExportData(exports),
		packages.WithOnExportOnly(func(e packages.ExportOnly) { exportOnly = append(exportOnly, e) }),
	).Load(&packages.Config{}, "./p")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	dep := pkgs[0].Imports["fakefmt"]
	require.NotNil(t, dep)
	assert.NotNil(t, dep.Types.Scope().Lookup("Stamp"), "the types are still there")
	assert.Empty(t, dep.Syntax, "but nothing said anything about them")

	require.Len(t, exportOnly, 1, "which is announced rather than left to be noticed in the output")
	assert.Equal(t, "fakefmt", exportOnly[0].Path)
	assert.Contains(t, exportOnly[0].Reason, "not on the filesystem")
}

// commentsOf flattens every comment in a set of files.
func commentsOf(files []*ast.File) string {
	var b bytes.Buffer
	for _, f := range files {
		for _, g := range f.Comments {
			b.WriteString(g.Text())
		}
	}

	return b.String()
}
