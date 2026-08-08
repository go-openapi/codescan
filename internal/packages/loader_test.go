// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package packages_test

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-openapi/codescan/internal/packages"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
	"golang.org/x/tools/go/gcexportdata"
)

// tagTree is a module whose files are selected by build constraints in every way go/build supports: a //go:build
// expression, its negation, a conjunction, and GOOS filename suffixes.
//
// It is an fstest.MapFS rather than testdata on disk so the test says something about WithFS at the same time —
// nothing here is reachable through the os package.
func tagTree() fstest.MapFS {
	return fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/tagtree\n\ngo 1.25.0\n")},
		"pkg/always.go": &fstest.MapFile{Data: []byte(
			"package pkg\n\ntype Always struct{ A string }\n")},
		"pkg/tagged.go": &fstest.MapFile{Data: []byte(
			"//go:build integration\n\npackage pkg\n\ntype Tagged struct{ B string }\n")},
		"pkg/negated.go": &fstest.MapFile{Data: []byte(
			"//go:build !integration\n\npackage pkg\n\ntype Negated struct{ C string }\n")},
		"pkg/combo.go": &fstest.MapFile{Data: []byte(
			"//go:build integration && custom\n\npackage pkg\n\ntype Combo struct{ D string }\n")},
		"pkg/impl_linux.go": &fstest.MapFile{Data: []byte(
			"package pkg\n\ntype OnLinux struct{ E string }\n")},
		"pkg/impl_windows.go": &fstest.MapFile{Data: []byte(
			"package pkg\n\ntype OnWindows struct{ F string }\n")},
	}
}

// loadedFiles returns the base names of the files the loader decided the package is made of.
//
// Every fixture in this file puts its package in ./pkg, so the pattern is fixed rather than a parameter: what varies
// between cases is the config, which is the thing under test.
func loadedFiles(t *testing.T, loader *packages.Loader, cfg *packages.Config) []string {
	t.Helper()

	pkgs, err := loader.Load(cfg, "./pkg")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	names := make([]string, 0, len(pkgs[0].GoFiles))
	for _, f := range pkgs[0].GoFiles {
		names = append(names, filepath.Base(f))
	}
	sort.Strings(names)

	return names
}

func TestLoadBuildTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		flags []string
		want  []string
	}{
		{
			name: "no tags: the negation is in, the tagged files are out",
			want: []string{"always.go", "impl_linux.go", "negated.go"},
		},
		{
			name:  "-tags integration flips the pair, conjunction still unmet",
			flags: []string{"-tags", "integration"},
			want:  []string{"always.go", "impl_linux.go", "tagged.go"},
		},
		{
			name:  "an unrelated tag changes nothing",
			flags: []string{"-tags", "custom"},
			want:  []string{"always.go", "impl_linux.go", "negated.go"},
		},
		{
			name:  "both tags satisfy the conjunction",
			flags: []string{"-tags", "integration,custom"},
			want:  []string{"always.go", "combo.go", "impl_linux.go", "tagged.go"},
		},
		{
			name:  "-tags=x= spelling is accepted too",
			flags: []string{"-tags=integration"},
			want:  []string{"always.go", "impl_linux.go", "tagged.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Pinned to linux/amd64 so the expectations describe the build constraints under test and not the platform the test
			// happens to run on.
			loader := packages.NewLoader(packages.WithFS(tagTree()), packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}))
			got := loadedFiles(t, loader, &packages.Config{BuildFlags: tt.flags})

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		goos   string
		goarch string
		want   []string
	}{
		{
			name: "linux picks the linux file",
			goos: "linux", goarch: "amd64",
			want: []string{"always.go", "impl_linux.go", "negated.go"},
		},
		{
			name: "windows picks the windows file",
			goos: "windows", goarch: "amd64",
			want: []string{"always.go", "impl_windows.go", "negated.go"},
		},
		{
			// The case the option exists for: a platform that matches no suffix in the tree drops both implementation files.
			// Running inside a WASI guest, this is what the default would do.
			name: "a platform with no matching file drops both",
			goos: "wasip1", goarch: "wasm",
			want: []string{"always.go", "negated.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader := packages.NewLoader(packages.WithFS(tagTree()), packages.WithGoEnv(packages.GoEnv{GOOS: tt.goos, GOARCH: tt.goarch}))
			got := loadedFiles(t, loader, &packages.Config{})

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadTargetPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("defaults to the running platform", func(t *testing.T) {
		t.Parallel()

		loader := packages.NewLoader(packages.WithFS(tagTree()))
		got := loadedFiles(t, loader, &packages.Config{})

		// Whatever the host is, the file for the *other* platform must not be in.
		if runtime.GOOS == "windows" {
			assert.Contains(t, got, "impl_windows.go")
			assert.NotContains(t, got, "impl_linux.go")
		} else {
			assert.NotContains(t, got, "impl_windows.go")
		}
	})

	t.Run("Config.Env overrides the running platform", func(t *testing.T) {
		t.Parallel()

		loader := packages.NewLoader(packages.WithFS(tagTree()))
		got := loadedFiles(t, loader, &packages.Config{Env: []string{"GOOS=windows", "GOARCH=amd64"}})

		assert.Contains(t, got, "impl_windows.go")
		assert.NotContains(t, got, "impl_linux.go")
	})

	t.Run("WithGoEnv wins over Config.Env", func(t *testing.T) {
		t.Parallel()

		loader := packages.NewLoader(packages.WithFS(tagTree()), packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}))
		got := loadedFiles(t, loader, &packages.Config{Env: []string{"GOOS=windows", "GOARCH=amd64"}})

		assert.Contains(t, got, "impl_linux.go")
		assert.NotContains(t, got, "impl_windows.go")
	})

	t.Run("a half-specified target leaves the other half alone", func(t *testing.T) {
		t.Parallel()

		loader := packages.NewLoader(packages.WithFS(tagTree()), packages.WithGoEnv(packages.GoEnv{GOOS: "windows", GOARCH: ""}))
		got := loadedFiles(t, loader, &packages.Config{})

		assert.Contains(t, got, "impl_windows.go")
	})
}

func TestLoadRecursivePattern(t *testing.T) {
	t.Parallel()

	tree := tagTree()
	tree["pkg/sub/sub.go"] = &fstest.MapFile{Data: []byte("package sub\n\ntype Sub struct{ A string }\n")}
	tree["pkg/testdata/skipped.go"] = &fstest.MapFile{Data: []byte("package testdata\n\ntype Skipped struct{}\n")}

	loader := packages.NewLoader(packages.WithFS(tree), packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}))
	pkgs, err := loader.Load(&packages.Config{}, "./...")
	require.NoError(t, err)

	paths := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		paths = append(paths, p.PkgPath)
	}
	sort.Strings(paths)

	// testdata is not a package as far as the go tool is concerned, and must not be one here either.
	assert.Equal(t, []string{"example.com/tagtree/pkg", "example.com/tagtree/pkg/sub"}, paths)
}

func TestLoadResolvesIntraModuleImports(t *testing.T) {
	t.Parallel()

	tree := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/m\n\ngo 1.25.0\n")},
		"a/a.go": &fstest.MapFile{Data: []byte(
			"package a\n\nimport \"example.com/m/b\"\n\ntype A struct{ B b.B }\n")},
		"b/b.go": &fstest.MapFile{Data: []byte("package b\n\ntype B struct{ Name string }\n")},
	}

	loader := packages.NewLoader(packages.WithFS(tree), packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}))
	pkgs, err := loader.Load(&packages.Config{}, "./a")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	require.NotNil(t, pkg.Types)
	assert.Empty(t, pkg.Errors, "the import should resolve from source, leaving no type errors")

	// The field's type must be the real b.B, not a stub: a stub has no fields.
	obj := pkg.Types.Scope().Lookup("A")
	require.NotNil(t, obj)
	assert.Contains(t, obj.Type().Underlying().String(), "example.com/m/b.B")
}

// TestLoadChecksFunctionBodies guards a regression that the fixture corpus caught: codescan discovers annotated types
// declared inside function bodies, so the type-checker must not skip them.
func TestLoadChecksFunctionBodies(t *testing.T) {
	t.Parallel()

	tree := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/m\n\ngo 1.25.0\n")},
		"p/p.go": &fstest.MapFile{Data: []byte(
			"package p\n\nfunc Outer() {\n\t// swagger:response inner\n\ttype Inner struct{ A string }\n\t_ = Inner{}\n}\n")},
	}

	loader := packages.NewLoader(packages.WithFS(tree), packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}))
	pkgs, err := loader.Load(&packages.Config{}, "./p")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	var found bool
	for ident, obj := range pkgs[0].TypesInfo.Defs {
		if ident.Name == "Inner" && obj != nil {
			found = true
		}
	}
	assert.True(t, found, "a type declared inside a function body must appear in TypesInfo.Defs")
}

func TestLoadReportsSynthesizedImports(t *testing.T) {
	t.Parallel()

	tree := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/m\n\ngo 1.25.0\n")},
		"p/p.go": &fstest.MapFile{Data: []byte(
			"package p\n\nimport (\n\t\"time\"\n\t\"example.com/nowhere/x\"\n)\n\n" +
				"type T struct {\n\tA time.Time\n\tB x.Thing\n}\n")},
	}

	var got []packages.Synthesized
	loader := packages.NewLoader(
		packages.WithFS(tree),
		packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}),
		packages.WithStubbedStdlib(),
		packages.WithOnSynthesized(func(s packages.Synthesized) { got = append(got, s) }),
	)

	_, err := loader.Load(&packages.Config{}, "./p")
	require.NoError(t, err)

	byPath := make(map[string]packages.Synthesized, len(got))
	for _, s := range got {
		byPath[s.Path] = s
	}

	// The standard library was withheld on purpose; the other import simply does not exist.
	// The caller needs to tell those apart, because only one of them is a mistake.
	stdlib, ok := byPath["time"]
	require.True(t, ok, "expected a report for the withheld standard library, got %v", byPath)
	assert.True(t, stdlib.Deliberate)

	missing, ok := byPath["example.com/nowhere/x"]
	require.True(t, ok, "expected a report for the unresolvable import, got %v", byPath)
	assert.False(t, missing.Deliberate)

	assert.Equal(t, "p/p.go", missing.Pos.Filename, "the report should point at the import that caused it")
	assert.Positive(t, missing.Pos.Line)
}

// TestLoadKeepsRootedPathsUnderRecursivePatterns guards the boundary between the caller's paths and io/fs's own rooted
// namespace: a walk must hand back paths in the form it was given, or every token.Position the scan reports comes out
// unresolvable.
func TestLoadKeepsRootedPathsUnderRecursivePatterns(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.com/m\n\ngo 1.25.0\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sub"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sub", "s.go"),
		[]byte("package sub\n\ntype S struct{ A string }\n"), 0o600))

	// The FS is rooted at the top of whatever holds the temp tree — "/" on a POSIX system, the volume elsewhere — because
	// a rooted path is exactly what has to survive the round trip through io/fs's own namespace.
	fsRoot := filepath.VolumeName(root) + string(filepath.Separator)

	loader := packages.NewLoader(packages.WithFS(os.DirFS(fsRoot)), packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}))
	pkgs, err := loader.Load(&packages.Config{Dir: root}, "./...")
	require.NoError(t, err)
	require.NotEmpty(t, pkgs)

	for _, p := range pkgs {
		for _, f := range p.GoFiles {
			assert.True(t, filepath.IsAbs(f),
				"file %q lost its root; positions derived from it would not resolve", f)
			assert.True(t, strings.HasPrefix(f, root),
				"file %q was not reported under the directory that was scanned", f)
		}
	}
}

// TestLoadReadsFromExportData feeds the loader export data it generated itself, so the test needs no toolchain and no
// GOROOT.
//
// The package stands in for any dependency: export data applies to everything outside the module being scanned.
func TestLoadReadsFromExportData(t *testing.T) {
	t.Parallel()

	// Type-check a stand-in for a standard-library package, then write it out the way the compiler would have.
	fset := token.NewFileSet()
	src := "package faketime\n\ntype Moment struct{ Seconds int64 }\n\nfunc (Moment) Marshal() string { return \"\" }\n"
	f, err := parser.ParseFile(fset, "faketime.go", src, 0)
	require.NoError(t, err)

	conf := types.Config{Importer: nil}
	pkg, err := conf.Check("faketime", fset, []*ast.File{f}, nil)
	require.NoError(t, err)

	var blob bytes.Buffer
	require.NoError(t, gcexportdata.Write(&blob, fset, pkg))

	exports := fstest.MapFS{"faketime.export": &fstest.MapFile{Data: blob.Bytes()}}
	tree := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/m\n\ngo 1.25.0\n")},
		"p/p.go": &fstest.MapFile{Data: []byte(
			"package p\n\nimport \"faketime\"\n\ntype T struct{ M faketime.Moment }\n")},
	}

	var synthesized []string
	loader := packages.NewLoader(
		packages.WithFS(tree),
		packages.WithGoEnv(packages.GoEnv{GOOS: "linux", GOARCH: "amd64"}),
		packages.WithExportData(exports),
		packages.WithOnSynthesized(func(s packages.Synthesized) { synthesized = append(synthesized, s.Path) }),
	)

	pkgs, err := loader.Load(&packages.Config{}, "./p")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	assert.NotContains(t, synthesized, "faketime",
		"a package served from export data must not be synthesized")

	obj := pkgs[0].Types.Scope().Lookup("T")
	require.NotNil(t, obj)

	// The decisive check: the imported type carries real structure.
	// A synthesized stand-in would have no fields and no methods, which is exactly the fidelity export data exists to
	// restore.
	outer, ok := obj.Type().Underlying().(*types.Struct)
	require.True(t, ok, "expected a struct, got %T", obj.Type().Underlying())

	named, ok := outer.Field(0).Type().(*types.Named)
	require.True(t, ok, "expected a named type, got %T", outer.Field(0).Type())
	assert.Equal(t, "faketime.Moment", named.String())

	inner, ok := named.Underlying().(*types.Struct)
	require.True(t, ok, "expected a struct, got %T", named.Underlying())
	assert.Equal(t, 1, inner.NumFields(), "fields should survive")
	assert.Equal(t, 1, named.NumMethods(), "the method set should survive")
}
