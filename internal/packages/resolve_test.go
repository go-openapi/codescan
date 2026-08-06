// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package packages_test

import (
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-openapi/codescan/internal/packages"
	"github.com/go-openapi/codescan/internal/packages/list"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// These cover pattern resolution, which the corpus A/B structurally cannot: every fixture scan is a single module
// scanned with ".", so nothing there exercises what a pattern *means*.

// writeTree materialises a map of path -> content under a fresh directory.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o750))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	}

	return root
}

// pkgPaths loads a pattern and returns the resulting import paths, sorted.
func pkgPaths(t *testing.T, dir, pattern string, opts ...packages.Option) []string {
	t.Helper()

	pkgs, err := packages.NewLoader(opts...).Load(&packages.Config{Dir: dir}, pattern)
	require.NoError(t, err)

	out := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		out = append(out, p.PkgPath)
	}
	sort.Strings(out)

	return out
}

// nestedTree is a main module with an unrelated module nested inside it.
//
// The nested module's path deliberately shares no prefix with the main one, so an import path derived from the wrong
// module is unmistakable rather than plausible.
func nestedTree(t *testing.T) string {
	t.Helper()

	return writeTree(t, map[string]string{
		"go.mod":        "module example.com/main\n\ngo 1.25.0\n",
		"a/a.go":        "package a\n\ntype A struct{}\n",
		"sub/go.mod":    "module totally.different/thing\n\ngo 1.25.0\n",
		"sub/b/b.go":    "package b\n\ntype B struct{}\n",
		"testdata/t.go": "package testdata\n",
	})
}

// A `...` pattern stops at a module boundary, as it does for the go command.
//
// Before this, the walk descended into every nested module it found — vendored trees, fixture corpora, sibling
// modules — and reported them all as packages of the main module.
// On this repo that turned 25 packages into 468.
func TestResolvePatterns_StopsAtNestedModule(t *testing.T) {
	t.Parallel()

	root := nestedTree(t)

	assert.Equal(t,
		pkgPaths(t, root, "./...", packages.WithStrategy(packages.StrategyGoPackages)),
		pkgPaths(t, root, "./...", packages.WithStrategy(packages.StrategyToolchainFree)),
		"a recursive pattern must match the same packages under either strategy")

	assert.Equal(t, []string{"example.com/main/a"},
		pkgPaths(t, root, "./...", packages.WithStrategy(packages.StrategyToolchainFree)))
}

// An import path is derived from the module that actually contains the directory.
//
// Naming a nested module's package explicitly is something the go command refuses ("directory is outside main module");
// we allow it, because scanning a subdirectory module directly is a reasonable thing to ask of a scanner.
// What must not happen is answering with a well-formed path that names a package which does not exist.
func TestResolvePatterns_ImportPathComesFromTheContainingModule(t *testing.T) {
	t.Parallel()

	root := nestedTree(t)

	assert.Equal(t, []string{"totally.different/thing/b"},
		pkgPaths(t, root, "./sub/b", packages.WithStrategy(packages.StrategyToolchainFree)),
		"the nested module declares this path; deriving it from the main module would invent one")
}

// A workspace makes a sibling module resolve to the copy being worked on.
//
// Missing this does not fail: the import is looked up in the module cache, missed, and synthesized, so the sibling's
// types arrive with no fields at all and the spec quietly thins out.
func TestResolveImport_WorkspaceSibling(t *testing.T) {
	t.Parallel()

	root := writeTree(t, map[string]string{
		"go.work":   "go 1.25.0\n\nuse (\n\t./m1\n\t./m2\n)\n",
		"m1/go.mod": "module example.com/m1\n\ngo 1.25.0\n",
		"m1/a/a.go": "package a\n\ntype T struct{ Field string }\n",
		"m2/go.mod": "module example.com/m2\n\ngo 1.25.0\n\nrequire example.com/m1 v0.0.0\n",
		"m2/b/b.go": "package b\n\nimport \"example.com/m1/a\"\n\ntype B struct{ A a.T }\n",
	})
	member := filepath.Join(root, "m2")

	fields := func(gowork string) int {
		pkgs, err := packages.NewLoader(
			packages.WithStrategy(packages.StrategyToolchainFree),
			packages.WithGoEnv(packages.GoEnv{GOWORK: gowork}),
		).Load(&packages.Config{Dir: member}, "./...")
		require.NoError(t, err)
		require.Len(t, pkgs, 1)

		dep := pkgs[0].Imports["example.com/m1/a"]
		if dep == nil || dep.Types == nil {
			return -1
		}
		obj := dep.Types.Scope().Lookup("T")
		if obj == nil {
			return -1
		}
		st, ok := obj.Type().Underlying().(*types.Struct)
		if !ok {
			return -1
		}

		return st.NumFields()
	}

	assert.Equal(t, 1, fields(""), "the sibling is read from its working copy, fields and all")
	assert.Equal(t, -1, fields("off"),
		"GOWORK=off is the caller disabling the workspace, so the sibling falls back to synthesis")
}

// GOWORK naming a file explicitly is honoured, so a caller can point at a workspace that does not sit above the
// directory being scanned.
func TestResolveImport_ExplicitGoWorkPath(t *testing.T) {
	t.Parallel()

	root := writeTree(t, map[string]string{
		"ws/go.work": "go 1.25.0\n\nuse (\n\t../m1\n\t../m2\n)\n",
		"m1/go.mod":  "module example.com/m1\n\ngo 1.25.0\n",
		"m1/a/a.go":  "package a\n\ntype T struct{ Field string }\n",
		"m2/go.mod":  "module example.com/m2\n\ngo 1.25.0\n\nrequire example.com/m1 v0.0.0\n",
		"m2/b/b.go":  "package b\n\nimport \"example.com/m1/a\"\n\ntype B struct{ A a.T }\n",
	})

	pkgs, err := packages.NewLoader(
		packages.WithStrategy(packages.StrategyToolchainFree),
		packages.WithGoEnv(packages.GoEnv{GOWORK: filepath.Join(root, "ws", "go.work")}),
	).Load(&packages.Config{Dir: filepath.Join(root, "m2")}, "./...")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	dep := pkgs[0].Imports["example.com/m1/a"]
	require.NotNil(t, dep)
	require.NotNil(t, dep.Types)
	assert.NotNil(t, dep.Types.Scope().Lookup("T"),
		"a go.work outside the scanned tree still governs it when named")
}

// A wildcard never reaches inside a vendor directory, but a package that merely happens to be called vendor is an
// ordinary match.
//
// The rule is about the wildcard, not about the name.
func TestResolvePatterns_VendorAndWildcards(t *testing.T) {
	t.Parallel()

	root := writeTree(t, map[string]string{
		"go.mod":            "module example.com/main\n\ngo 1.25.0\n",
		"a/a.go":            "package a\n",
		"vendor/v.go":       "package vendored\n",
		"vendor/deep/d.go":  "package deep\n",
		"sub/vendor/x/x.go": "package x\n",
	})

	assert.Equal(t,
		pkgPaths(t, root, "./...", packages.WithStrategy(packages.StrategyGoPackages)),
		pkgPaths(t, root, "./...", packages.WithStrategy(packages.StrategyToolchainFree)))

	got := pkgPaths(t, root, "./...", packages.WithStrategy(packages.StrategyToolchainFree))
	assert.Contains(t, got, "example.com/main/vendor", "the directory itself is a package like any other")
	assert.NotContains(t, got, "example.com/main/vendor/deep", "but a wildcard does not reach inside it")
	assert.NotContains(t, got, "example.com/main/sub/vendor/x", "at any depth")
}

// `...` is a wildcard anywhere, not only as a whole trailing path element.
//
// Both of these used to be rejected outright.
func TestResolvePatterns_WildcardForms(t *testing.T) {
	t.Parallel()

	root := writeTree(t, map[string]string{
		"go.mod":                 "module example.com/main\n\ngo 1.25.0\n",
		"internal/pkg/a.go":      "package pkg\n",
		"internal/other/b.go":    "package other\n",
		"internal/deep/pkg/c.go": "package pkg\n",
	})

	for _, pattern := range []string{
		"./internal/pk...",   // partial segment
		"./internal/.../pkg", // mid-path
		"./internal/...",     // ordinary suffix
	} {
		t.Run(pattern, func(t *testing.T) {
			assert.Equal(t,
				pkgPaths(t, root, pattern, packages.WithStrategy(packages.StrategyGoPackages)),
				pkgPaths(t, root, pattern, packages.WithStrategy(packages.StrategyToolchainFree)))
		})
	}
}

// A vendor directory is authoritative only when `go mod vendor` wrote modules.txt, and -mod=mod turns it off again.
//
// Reading it on the strength of the directory alone is how a stale copy silently replaced the real dependency.
func TestResolveImport_VendorIsAuthoritativeOnlyWithModulesTxt(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"go.mod":                      "module example.com/main\n\ngo 1.25.0\n\nrequire example.com/dep v1.0.0\n",
		"a/a.go":                      "package a\n\nimport \"example.com/dep\"\n\nvar _ = dep.Sentinel\n",
		"vendor/example.com/dep/d.go": "package dep\n\nconst Sentinel = true\n",
	}

	// Without modules.txt the tree is not a vendored build, so the import must not resolve from it.
	bare := writeTree(t, files)
	assert.False(t, readsVendoredSentinel(t, &packages.Config{Dir: bare}),
		"a directory named vendor is not by itself a vendored build")

	// With it, the vendored copy is what the build reads.
	files["vendor/modules.txt"] = "# example.com/dep v1.0.0\n## explicit; go 1.25\nexample.com/dep\n"
	vendored := writeTree(t, files)
	assert.True(t, readsVendoredSentinel(t, &packages.Config{Dir: vendored}),
		"modules.txt is what makes vendor authoritative")

	// -mod=mod is the explicit opt-out.
	assert.False(t, readsVendoredSentinel(t, &packages.Config{
		Dir: vendored, BuildFlags: []string{"-mod=mod"},
	}), "-mod=mod ignores the vendor directory")
}

// readsVendoredSentinel reports whether ./a's dependency resolved to the vendored copy, which is the only copy carrying
// Sentinel.
func readsVendoredSentinel(t *testing.T, cfg *packages.Config) bool {
	t.Helper()

	pkgs, err := packages.NewLoader(packages.WithStrategy(packages.StrategyToolchainFree)).Load(cfg, "./a")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	dep := pkgs[0].Imports["example.com/dep"]
	if dep == nil || dep.Types == nil {
		return false
	}

	return dep.Types.Scope().Lookup("Sentinel") != nil
}

// A version-pinned replace applies to that version and no other.
//
// The two forms look almost identical, and applying the pinned one regardless swaps in a substitute the build never
// asked for — quietly, since the substitute usually compiles.
func TestReadRequirements_ReplaceRespectsVersions(t *testing.T) {
	t.Parallel()

	tree := func(replaceLine, requireVersion string) string {
		return writeTree(t, map[string]string{
			"go.mod": "module example.com/main\n\ngo 1.25.0\n\nrequire example.com/dep " + requireVersion +
				"\n\n" + replaceLine + "\n",
			"a/a.go":       "package a\n\nimport \"example.com/dep\"\n\nvar _ = dep.LocalSentinel\n",
			"local/go.mod": "module example.com/dep\n\ngo 1.25.0\n",
			"local/d.go":   "package dep\n\nconst LocalSentinel = true\n",
		})
	}

	applied := func(root string) bool {
		pkgs, err := packages.NewLoader(packages.WithStrategy(packages.StrategyToolchainFree)).
			Load(&packages.Config{Dir: root}, "./a")
		require.NoError(t, err)
		require.Len(t, pkgs, 1)

		dep := pkgs[0].Imports["example.com/dep"]

		return dep != nil && dep.Types != nil && dep.Types.Scope().Lookup("LocalSentinel") != nil
	}

	assert.True(t, applied(tree("replace example.com/dep => ./local", "v1.2.0")),
		"an unversioned replace applies to every version")
	assert.True(t, applied(tree("replace example.com/dep v1.0.0 => ./local", "v1.0.0")),
		"a pinned replace applies to the version it names")
	assert.False(t, applied(tree("replace example.com/dep v1.0.0 => ./local", "v1.2.0")),
		"and to no other")
}

// A go.mod that cannot be read fails the load.
//
// Degrading was worse than it sounds: with no requirement placed, every dependency in the module falls through to
// synthesis, so the caller gets a wall of synthesized-import warnings and no mention of the one line that caused them.
func TestReadRequirements_InvalidGoModIsFatal(t *testing.T) {
	t.Parallel()

	root := writeTree(t, map[string]string{
		// v2 without a /v2 path suffix is not a legal requirement.
		"go.mod": "module example.com/main\n\ngo 1.25.0\n\nrequire example.com/dep v2.0.0\n",
		"a/a.go": "package a\n",
	})

	_, err := packages.NewLoader(packages.WithStrategy(packages.StrategyToolchainFree)).
		Load(&packages.Config{Dir: root}, "./a")

	require.Error(t, err)
	assert.ErrorIs(t, err, list.ErrInvalidGoMod)
	assert.Contains(t, err.Error(), "go.mod", "the message names the file at fault")
}

// A tree with no module at all still scans, and must not name its packages after the machine that scanned them: a
// package path reaches the emitted spec through x-go-package.
func TestPkgPath_NoModuleDoesNotLeakAbsolutePaths(t *testing.T) {
	t.Parallel()

	root := writeTree(t, map[string]string{"a/a.go": "package a\n\ntype A struct{}\n"})

	got := pkgPaths(t, root, "./a", packages.WithStrategy(packages.StrategyToolchainFree))

	require.Len(t, got, 1)
	assert.Equal(t, "a", got[0])
	assert.NotContains(t, got[0], root, "the scanning machine's layout is not part of the answer")
}
