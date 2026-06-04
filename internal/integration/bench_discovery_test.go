// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

// Axis F — discovery cost, separated from loading.
//
// Finding every annotation in a closure and loading that closure are the same operation today. They
// need not be: `go list` gives the package graph and every package's GoFiles without types or
// syntax, a byte scan over those files rejects ~95% of them, and only the survivors are worth
// parsing. This measures each of those three steps on its own.
//
// Discovery has to stay EXHAUSTIVE — stdlib and all dependencies included — which is why it is
// worth making cheap rather than making narrow. Discriminator subtypes are discovered backwards (a
// subtype $refs its base and nothing $refs the subtype), so an annotated type in a dependency is
// never reached by following types from the root packages.
//
// See bench_scanner_axes.md.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/go-openapi/testify/v2/require"
)

// annotationNeedle is the byte-level pre-filter, and is also the exact test the loader's own
// per-dependency export-data policy uses (internal/packages.scanForAnnotations).
//
// It can only over-admit: a `swagger:` inside a string literal, or inside prose ABOUT annotations,
// costs one wasted parse. It cannot under-admit, because a real annotation contains this literal by
// definition — which is what makes the cheap pass safe to run INSTEAD of loading the package.
const annotationNeedle = "swagger:"

// isAnnotationLine is a regexp-free restatement of parsers.rxCommentPrefix followed by the keyword:
// strip the leading comment noise, allow one markdown table pipe, and require `swagger:` to START
// what remains.
//
// The line-start rule is the whole subtlety. A naive substring test counts 14 annotated packages on
// dockerctl where this counts 2 — codescan's own godoc, read as a dependency, discusses `swagger:`
// constantly. The gap between the two rules is measured directly by Axis H, because the loader
// routes dependencies on the naive rule and pays a full source load for each over-admission.
func isAnnotationLine(text string) bool {
	i := 0
	for i < len(text) && strings.ContainsRune(" \t/*-", rune(text[i])) {
		i++
	}
	if i < len(text) && text[i] == '|' {
		i++
		for i < len(text) && text[i] == ' ' {
			i++
		}
	}

	return strings.HasPrefix(text[i:], annotationNeedle)
}

// fileList is the cheapest useful load: the package graph and every package's GoFiles, with no
// types and no syntax.
//
// It is also what makes on-demand syntax possible later — GoFiles is populated for the whole
// closure here, so materialising a package's source afterwards is plain parser.ParseFile on known
// paths, with no second `go list`.
func fileList(corpus benchCorpus) (map[string]*packages.Package, time.Duration, error) {
	const mode = packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps

	start := time.Now()
	pkgs, err := packages.Load(&packages.Config{Dir: corpus.dir, Mode: mode}, corpus.pattern)
	if err != nil {
		return nil, 0, err
	}

	return walkGraph(pkgs), time.Since(start), nil
}

// annotatedDecls is the annotation index for one package: the type names whose doc comment carries
// an annotation.
type annotatedDecls struct {
	pkgPath string
	names   []string
}

// discoveryResult is what the two discovery passes produce: the index itself, the two package sets
// the two admission rules yield, and what each pass cost.
type discoveryResult struct {
	// annotated holds the packages with at least one annotated TYPE declaration, and their names.
	// These are the roots the working-set walk starts from (Axis G).
	annotated []annotatedDecls

	// markerPkgs holds the packages admitted by the naive substring rule — the loader's own
	// per-dependency test. A superset of realPkgs, and the difference is what Axis H prices.
	markerPkgs map[string]bool

	// realPkgs holds the packages with at least one comment line that genuinely STARTS an
	// annotation, anywhere in the package (not only above a type declaration, so routes and meta
	// blocks count too).
	realPkgs map[string]bool

	files       int
	survivors   int
	sourceBytes int64
	prefilter   time.Duration
	parse       time.Duration
}

// discoverAnnotations runs the two cheap discovery passes over every file in the closure and
// returns the annotated declarations it found, plus the cost of each pass.
//
// Deliberately exhaustive: see the file comment on why discovery cannot be demand-driven.
func discoverAnnotations(all map[string]*packages.Package) discoveryResult {
	type fileRef struct{ pkgPath, path string }

	res := discoveryResult{markerPkgs: map[string]bool{}, realPkgs: map[string]bool{}}

	var files []fileRef
	for pkgPath, p := range all {
		for _, f := range p.GoFiles {
			files = append(files, fileRef{pkgPath, f})
		}
	}
	res.files = len(files)

	// Pass 1 — read every file, keep the ones containing the literal.
	start := time.Now()
	// A starting capacity, sized from the ~5% hit rate the prefilter has on both corpora.
	const expectedHitRate = 16

	kept := make([]fileRef, 0, len(files)/expectedHitRate)
	contents := make(map[string][]byte, len(files)/expectedHitRate)
	for _, fr := range files {
		blob, err := os.ReadFile(fr.path)
		if err != nil {
			continue
		}
		res.sourceBytes += int64(len(blob))
		if bytes.Contains(blob, []byte(annotationNeedle)) {
			kept = append(kept, fr)
			contents[fr.path] = blob
			res.markerPkgs[fr.pkgPath] = true
		}
	}
	res.prefilter = time.Since(start)
	res.survivors = len(kept)

	// Pass 2 — parse only the survivors and apply the real, line-start rule.
	start = time.Now()
	fset := token.NewFileSet()
	byPkg := map[string][]string{}
	for _, fr := range kept {
		af, err := parser.ParseFile(fset, fr.path, contents[fr.path], parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		if fileCarriesAnnotation(af) {
			res.realPkgs[fr.pkgPath] = true
		}
		// Guarded: a file can survive the prefilter on a `swagger:` mention in prose and yield no
		// annotated decl at all. An unguarded append would still create the map key and count the
		// package as a root.
		if names := annotatedTypeNames(af); len(names) > 0 {
			byPkg[fr.pkgPath] = append(byPkg[fr.pkgPath], names...)
		}
	}
	res.parse = time.Since(start)

	for pkgPath, names := range byPkg {
		res.annotated = append(res.annotated, annotatedDecls{pkgPath: pkgPath, names: names})
	}
	sort.Slice(res.annotated, func(i, j int) bool { return res.annotated[i].pkgPath < res.annotated[j].pkgPath })

	return res
}

// fileCarriesAnnotation reports whether any comment in af genuinely starts an annotation.
//
// Every comment, not just doc comments above type declarations: `swagger:route` and `swagger:meta`
// are free-standing, and a package carrying only those still has to be read from source.
func fileCarriesAnnotation(af *ast.File) bool {
	for _, group := range af.Comments {
		for _, c := range group.List {
			if isAnnotationLine(c.Text) {
				return true
			}
		}
	}

	return false
}

// annotatedTypeNames returns the type names in af whose doc comment carries an annotation line.
func annotatedTypeNames(af *ast.File) []string {
	var names []string
	for _, d := range af.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, sp := range gd.Specs {
			ts, ok := sp.(*ast.TypeSpec)
			if !ok {
				continue
			}
			doc := ts.Doc
			if doc == nil {
				doc = gd.Doc
			}
			if doc == nil {
				continue
			}
			for _, c := range doc.List {
				if isAnnotationLine(c.Text) {
					names = append(names, ts.Name.Name)

					break
				}
			}
		}
	}

	return names
}

// TestBenchDiscovery is Axis F: what a complete annotation index over the whole closure costs when
// nothing is loaded to build it.
//
// The figure to compare it against is the first rung of Axis D — today's load, which produces the
// same index as a side effect of type-checking everything.
func TestBenchDiscovery(t *testing.T) {
	requireBenchEnabled(t)

	for _, corpus := range benchCorpora(t) {
		t.Run(corpus.name, func(t *testing.T) {
			base := allocBaseline()

			all, listed, err := fileList(corpus)
			require.NoError(t, err, "go list")

			res := discoverAnnotations(all)

			t.Logf("=== discovery without loading — %s ===", corpus.name)
			t.Logf("go list (graph + GoFiles)                %7.2fs  graph=%d packages, %d Go files, %.1f MB of source",
				listed.Seconds(), len(all), res.files, float64(res.sourceBytes)/bytesPerMB)
			t.Logf("byte prefilter over ALL files            %7.2fs  -> %d/%d files survive (%.1f%%)",
				res.prefilter.Seconds(), res.survivors, res.files, percent(res.survivors, res.files))
			t.Logf("parse + classify survivors only          %7.2fs  -> %d packages with an annotated type decl",
				res.parse.Seconds(), len(res.annotated))
			t.Logf("TOTAL discovery                          %7.2fs  allocated=%.0f MB (transient), retained=%.0f MB",
				(listed + res.prefilter + res.parse).Seconds(), mbOf(allocatedSince(base)), mbOf(retainedHeap()))
			t.Logf("admission rules: substring=%d packages, line-start=%d packages",
				len(res.markerPkgs), len(res.realPkgs))

			for _, a := range res.annotated {
				t.Logf("    %-70s %d annotated type decls", a.pkgPath, len(a.names))
			}
		})
	}
}

// percent renders n/total as a percentage, tolerating a zero total.
func percent(n, total int) float64 {
	if total == 0 {
		return 0
	}

	return 100 * float64(n) / float64(total)
}
