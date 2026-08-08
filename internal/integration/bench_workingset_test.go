// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

// Axis G — the working set, and Axis H — what the per-dependency export-data policy leaves on the
// table.
//
// Axis G asks how much of a loaded closure anything ever needs: follow types outward from the
// annotated declarations and count the PACKAGES reached. The annotated set is not the working set —
// a package carrying no annotation at all is pulled in purely because a field is typed that way.
//
// Axis H prices the gap between three sets of dependencies, which under the per-dependency
// export-data policy is where the remaining cost lives:
//
//	all           every dependency in the closure
//	substring     those whose source contains the literal `swagger:` — the loader's own admission
//	              rule, and therefore what it loads from source today
//	line-start    those that genuinely carry an annotation (parsers.rxCommentPrefix's rule)
//	reached       those the type-reference closure actually lands in
//
// substring ⊇ line-start, and reached is a different set again — it is neither a subset nor a
// superset, since a package can be reached by a field type while saying nothing itself. What the
// harness reports is the cost of parsing each set's source, so the gap between "what is loaded" and
// "what is needed" is a number rather than an argument.
//
// See bench_scanner_axes.md.

import (
	"fmt"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"sort"
	"testing"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/go-openapi/testify/v2/require"
)

// maxFollowDepth bounds the structural walk. Recursive types are cut by the seen-set, not by this;
// the bound only guards pathological nesting.
const maxFollowDepth = 64

// typeCloser walks outward from a declaration the way a schema builder does, recording which
// PACKAGES the walk lands in. That set — not the annotated set — is the scanner's real working set.
type typeCloser struct {
	seenTypes map[string]bool
	touched   map[string]bool
}

func newTypeCloser() *typeCloser {
	return &typeCloser{seenTypes: map[string]bool{}, touched: map[string]bool{}}
}

// follow records t's declaring package and recurses through its structure.
//
// This APPROXIMATES the schema builder rather than replaying it: struct fields, element types, map
// keys and values, type arguments, interface methods and signature results. Good enough to size the
// closure; NOT good enough to prove an incremental loader reaches the same set — that is what the
// spec A/B in loader_agreement_test.go is for.
func (c *typeCloser) follow(t types.Type, depth int) {
	if t == nil || depth > maxFollowDepth {
		return
	}

	switch tt := t.(type) {
	case *types.Named:
		if obj := tt.Obj(); obj != nil && obj.Pkg() != nil {
			key := obj.Pkg().Path() + "." + obj.Name()
			if c.seenTypes[key] {
				return
			}
			c.seenTypes[key] = true
			c.touched[obj.Pkg().Path()] = true
		}
		for arg := range tt.TypeArgs().Types() {
			c.follow(arg, depth+1)
		}
		c.follow(tt.Underlying(), depth+1)
	case *types.Alias:
		if obj := tt.Obj(); obj != nil && obj.Pkg() != nil {
			c.touched[obj.Pkg().Path()] = true
		}
		c.follow(types.Unalias(tt), depth+1)
	case *types.Struct:
		for field := range tt.Fields() {
			c.follow(field.Type(), depth+1)
		}
	case *types.Pointer:
		c.follow(tt.Elem(), depth+1)
	case *types.Slice:
		c.follow(tt.Elem(), depth+1)
	case *types.Array:
		c.follow(tt.Elem(), depth+1)
	case *types.Map:
		c.follow(tt.Key(), depth+1)
		c.follow(tt.Elem(), depth+1)
	case *types.Interface:
		for method := range tt.Methods() {
			c.follow(method.Type(), depth+1)
		}
	case *types.Signature:
		if res := tt.Results(); res != nil {
			for v := range res.Variables() {
				c.follow(v.Type(), depth+1)
			}
		}
	}
}

// workingSet is one corpus measured: the loaded graph, the discovery result over it, and the
// type-reference closure from the annotated roots.
type workingSet struct {
	all       map[string]*packages.Package
	discovery discoveryResult
	reached   map[string]bool
	roots     int
}

// measureWorkingSet loads the closure with types but no syntax — the cheapest load the walk can
// follow — then discovers annotations and closes over the types they name.
func measureWorkingSet(tb testing.TB, corpus benchCorpus) workingSet {
	tb.Helper()

	const mode = packages.NeedName | packages.NeedFiles | packages.NeedImports |
		packages.NeedDeps | packages.NeedTypes

	pkgs, err := packages.Load(&packages.Config{Dir: corpus.dir, Mode: mode}, corpus.pattern)
	require.NoError(tb, err, "load")

	ws := workingSet{all: walkGraph(pkgs)}
	ws.discovery = discoverAnnotations(ws.all)

	closer := newTypeCloser()
	for _, a := range ws.discovery.annotated {
		p := ws.all[a.pkgPath]
		if p == nil || p.Types == nil {
			continue
		}
		closer.touched[a.pkgPath] = true
		for _, name := range a.names {
			obj := p.Types.Scope().Lookup(name)
			if obj == nil {
				continue
			}
			ws.roots++
			closer.follow(obj.Type(), 0)
		}
	}
	ws.reached = closer.touched

	return ws
}

// TestBenchWorkingSet is Axis G: how much of the loaded closure anything ever needs.
func TestBenchWorkingSet(t *testing.T) {
	requireBenchEnabled(t)

	for _, corpus := range benchCorpora(t) {
		t.Run(corpus.name, func(t *testing.T) {
			ws := measureWorkingSet(t, corpus)

			std, external := 0, 0
			reached := make([]string, 0, len(ws.reached))
			for p := range ws.reached {
				reached = append(reached, p)
				if isStdlibPath(p) {
					std++
				} else {
					external++
				}
			}
			sort.Strings(reached)

			t.Logf("=== the working set — %s ===", corpus.name)
			t.Logf("graph loaded                          %d packages", len(ws.all))
			t.Logf("annotated packages (roots)            %d  (%d annotated type decls)",
				len(ws.discovery.annotated), ws.roots)
			t.Logf("type-reference closure                %d packages  (%d non-stdlib, %d stdlib)",
				len(ws.reached), external, std)
			t.Logf("NEVER NEEDED                          %d / %d (%.1f%%)",
				len(ws.all)-len(ws.reached), len(ws.all),
				percent(len(ws.all)-len(ws.reached), len(ws.all)))

			annotated := make(map[string]bool, len(ws.discovery.annotated))
			for _, a := range ws.discovery.annotated {
				annotated[a.pkgPath] = true
			}
			for _, p := range reached {
				mark := "  " // reached only by following a type: invisible to the annotation prefilter
				if annotated[p] {
					mark = " *"
				}
				t.Logf("  %s %s", mark, p)
			}
		})
	}
}

// parseCost parses the source of a set of packages the way the loader would when it decides to read
// one, and reports what that costs.
//
// ParseComments is the point: comments are the only reason a dependency's source is worth reading
// at all, and they are also what makes parsing expensive.
func parseCost(all map[string]*packages.Package, want map[string]bool) (elapsed time.Duration, retained uint64, pkgs, files int) {
	base := retainedHeap()

	start := time.Now()
	fset := token.NewFileSet()
	parsed := make([]any, 0, len(want))
	for path := range want {
		p, ok := all[path]
		if !ok || len(p.GoFiles) == 0 {
			continue
		}
		pkgs++
		for _, name := range p.GoFiles {
			af, err := parser.ParseFile(fset, name, nil, parser.ParseComments|parser.SkipObjectResolution)
			if err != nil {
				continue
			}
			files++
			parsed = append(parsed, af)
		}
	}
	elapsed = time.Since(start)
	retained = retainedSince(base)

	runtime.KeepAlive(parsed)
	runtime.KeepAlive(fset)

	return elapsed, retained, pkgs, files
}

// TestBenchDependencyRouting is Axis H: what the per-dependency export-data policy loads from
// source, against what a scan actually reaches.
//
// The policy routes a dependency on the naive substring test, so it pays a full source load for
// every package that merely MENTIONS `swagger:` — codescan's own godoc among them. Two tighter sets
// exist: the packages that genuinely carry an annotation, and the packages the type walk reaches.
// The distance between them is what is still on the table, and on a corpus with a large annotated
// dependency it will not be small.
func TestBenchDependencyRouting(t *testing.T) {
	requireBenchEnabled(t)

	for _, corpus := range benchCorpora(t) {
		t.Run(corpus.name, func(t *testing.T) {
			ws := measureWorkingSet(t, corpus)

			everything := make(map[string]bool, len(ws.all))
			for path := range ws.all {
				everything[path] = true
			}

			t.Logf("=== dependency routing: what is parsed vs what is needed — %s ===", corpus.name)
			for _, set := range []struct {
				name string
				want map[string]bool
			}{
				{"all packages in the closure", everything},
				{"admitted by substring (the loader's rule)", ws.discovery.markerPkgs},
				{"admitted by line-start (a real annotation)", ws.discovery.realPkgs},
				{"reached by the type-reference closure", ws.reached},
			} {
				elapsed, retained, pkgs, files := parseCost(ws.all, set.want)
				reportRow(t, set.name, elapsed, retained,
					sprintfPkgFiles(pkgs, files, percent(pkgs, len(ws.all))))
			}
		})
	}
}

// sprintfPkgFiles renders the tail columns of an Axis H row.
func sprintfPkgFiles(pkgs, files int, share float64) string {
	return fmt.Sprintf("pkgs=%3d (%.1f%% of the closure)  files=%d", pkgs, share, files)
}
