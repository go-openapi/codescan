// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package packages

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
)

// attachAnnotatedDependencies gives a dependency its source back when that source has something to say.
//
// The two strategies arrive at the same policy from opposite ends, because they have opposite amounts of control.
// The toolchain-free one resolves imports itself, so it decides per dependency while the load is happening: a package
// whose source carries the marker is read from source, one that does not is taken from export data untouched. Here
// `go list` and go/packages own resolution, and the only lever is a LoadMode — one value for the whole load, with no
// hook to say "except this one". So the choice cannot be made during the load and is made after it: take every
// dependency from export data, then hand back the source of the few that were worth reading.
//
// What makes that possible is that the cheap load still says where the source IS. compiledDepsMode keeps NeedFiles,
// so a dependency comes back with GoFiles populated, types complete and no syntax — locatable, just unread. Parsing
// those files is the whole of the work; nothing is type-checked twice, because every declaration the source names is
// already an object in the export-data scope and the two halves are joined by name.
//
// The assembled shape — export-data types beside separately parsed syntax — carries no types.Info.Types, and cannot:
// its entries are unconstructible outside go/types. That used to rule this out altogether. The builders no longer
// read that map, and a spec builds identically without it, which is what makes the whole approach available. See
// [§annotated-dependencies](../scanner/README.md#annotated-dependencies).
func attachAnnotatedDependencies(roots []*Package, onExportOnly func(ExportOnly)) {
	seen := make(map[string]bool, len(roots))

	// One buffer for the whole walk. The marker scan holds nothing else, so the largest source file in the graph
	// costs what the smallest one does.
	var buf [annotationChunk]byte

	var walk func(*Package)
	walk = func(pkg *Package) {
		if pkg == nil || seen[pkg.ID] {
			return
		}
		seen[pkg.ID] = true

		attachSource(pkg, buf[:], onExportOnly)

		for _, imp := range pkg.Imports {
			walk(imp)
		}
	}

	for _, root := range roots {
		walk(root)
	}
}

// attachSource reads one package's source back onto it, or says why it did not.
//
// The three refusals are deliberately distinguished rather than collapsed into "no source": each is announced with
// its own reason, and the scanner replays that reason at the point some builder actually wanted the declaration.
// "Nothing in it is annotated" is the ordinary case and by far the most common — it is the policy working, not a
// fault, and it is worth recording only because a lookup landing there later has no other way to explain itself.
func attachSource(pkg *Package, buf []byte, onExportOnly func(ExportOnly)) {
	if pkg.Types == nil || len(pkg.Syntax) > 0 {
		// Loaded in the ordinary way: the roots, and anything else go list handed over whole.
		return
	}
	if pkg.Fset == nil {
		// Positions parsed into a private FileSet would not resolve against the ones the scan already holds, which
		// is worse than not attaching at all.
		announceExportOnly(onExportOnly, pkg.PkgPath, "the load left it with no position information")

		return
	}

	if len(pkg.GoFiles) == 0 {
		announceExportOnly(onExportOnly, pkg.PkgPath, "its source is not on the filesystem")

		return
	}

	if !filesCarryMarker(pkg.GoFiles, buf) {
		announceExportOnly(onExportOnly, pkg.PkgPath, "nothing in its source is annotated, so it was not parsed")

		return
	}

	syntax := parseFilesForComments(pkg.Fset, pkg.GoFiles)
	if len(syntax) == 0 {
		announceExportOnly(onExportOnly, pkg.PkgPath, "its source could not be parsed")

		return
	}

	pkg.Syntax = syntax
	pkg.CompiledGoFiles = pkg.GoFiles
	pkg.TypesInfo = bridgeDefs(syntax, pkg.Types)
}

// announceExportOnly reports a dependency whose types were read but whose source was not.
//
// Every package is visited once, so there is no dedup to do here.
func announceExportOnly(onExportOnly func(ExportOnly), importPath, why string) {
	if onExportOnly == nil {
		return
	}

	onExportOnly(ExportOnly{Path: importPath, Reason: why})
}

// filesCarryMarker reports whether any of a package's files contains the annotation marker.
//
// The real filesystem, unconditionally: these paths came from `go list`, which only ever runs against it.
func filesCarryMarker(paths []string, buf []byte) bool {
	for _, path := range paths {
		if fileCarriesMarker(openOSFile, path, buf) {
			return true
		}
	}

	return false
}

func openOSFile(path string) (io.ReadCloser, error) { return os.Open(path) }

// parseFilesForComments reads a package's source for what it says, not for what it means.
//
// No type-checking follows, so this is parsing alone — the cheap half — and the comments are the entire reason for
// doing it. Object resolution is skipped for the same reason: the objects are already in the export-data scope.
func parseFilesForComments(fset *token.FileSet, paths []string) []*ast.File {
	syntax := make([]*ast.File, 0, len(paths))

	for _, path := range paths {
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if f == nil {
			continue
		}
		_ = err // a partially parsed file still carries the declarations above the fault

		syntax = append(syntax, f)
	}

	return syntax
}

// bridgeDefs joins parsed declarations to the objects export data already holds.
//
// It is a name lookup rather than a type-check: a top-level declaration is in the package scope under exactly its own
// name, which is all TypesInfo.Defs is asked for here. Unexported names are absent from export data and stay
// unmapped, which is correct — nothing outside the package can refer to one.
func bridgeDefs(syntax []*ast.File, tpkg *types.Package) *types.Info {
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}}
	scope := tpkg.Scope()

	for _, f := range syntax {
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range gen.Specs {
				switch sp := spec.(type) {
				case *ast.TypeSpec:
					if obj := scope.Lookup(sp.Name.Name); obj != nil {
						info.Defs[sp.Name] = obj
					}
				case *ast.ValueSpec:
					for _, name := range sp.Names {
						if obj := scope.Lookup(name.Name); obj != nil {
							info.Defs[name] = obj
						}
					}
				}
			}
		}
	}

	return info
}
