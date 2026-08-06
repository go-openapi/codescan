// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package packages

import (
	"bytes"
	"fmt"
	"go/types"
	"io"
	"io/fs"
	"path"

	"golang.org/x/tools/go/gcexportdata"
)

// Reading a package from the compiler's export data.
//
// Type-checking the standard library from source is what a full scan spends nearly all its time on:
// 190 packages and 1195 files for a fixture as small as the petstore, and a WebAssembly guest pays
// a five to six-fold compute tax on top.
//
// None of that work is discovery — the answers were already computed when the toolchain built those packages,
// and the compiler wrote them down. So read them instead.
//
// The saving is in the parsing and type-checking avoided, not in the I/O: filesystem syscalls account for under 2%
// of a full WASI scan.

// exportedPackage builds the *Package a dependency served from export data is seen as.
//
// Types only, and no syntax.
//
// A package reached this way is one that says nothing about its own types — see carriesAnnotations — so there is
// nothing in its source for a scan to read.
func (ld *loadState) exportedPackage(importPath string, tpkg *types.Package) *Package {
	return &Package{
		ID:      importPath,
		Name:    tpkg.Name(),
		PkgPath: importPath,
		Types:   tpkg,
		Fset:    ld.fset,
		Imports: map[string]*Package{},
	}
}

// carriesAnnotations reports whether a dependency's source says anything a scan would read.
//
// This is the whole of the export-data policy, and it is a policy rather than an optimisation.
//
// Export data holds types and not comments, and there is no way to put the missing half back afterwards: go/types
// records what a type EXPRESSION denotes in types.Info.Types, whose entries cannot be constructed outside that package.
// The field distinguishing a type from a value is unexported.
//
// A package assembled from export data plus parsed syntax therefore has declarations the builders will read and no
// record of what those declarations denote, which is not a degraded scan but a panicking one.
//
// So the choice is made per package and made whole: a dependency that carries annotations is loaded from source like
// any other, and one that does not is taken from export data with no syntax at all.
//
// Nothing is lost either way, and the saving survives, because it was never in the packages a scan reads — it is in
// the closure behind them, which export data still serves.
func (ld *loadState) carriesAnnotations(importPath string) bool {
	if known, ok := ld.annotated[importPath]; ok {
		return known
	}

	found := ld.scanForAnnotations(importPath)
	ld.annotated[importPath] = found

	return found
}

// annotationMarker is what every codescan annotation begins with.
const annotationMarker = "swagger:"

// scanForAnnotations looks for the marker in a package's source, without parsing it.
//
// This scan should capture _at least_ what we need as it is an optimization. It doesn't have to resolve the
// exact regular expression in comments, just to discard all obvious unmatched content.
func (ld *loadState) scanForAnnotations(importPath string) bool {
	dir, _, ok := ld.res.ResolveImport(importPath)
	if !ok {
		// No source to consult.
		// The types still stand; what the package said about them is out of reach, and that is worth saying because it shows
		// up in the output as nothing at all.
		ld.reportExportOnly(importPath, "its source is not on the filesystem")

		return false
	}

	bp, err := ld.ctx.ImportDir(dir, 0)
	if err != nil {
		ld.reportExportOnly(importPath, "its source could not be read")

		return false
	}

	names := make([]string, 0, len(bp.GoFiles)+len(bp.CgoFiles))
	names = append(names, bp.GoFiles...)
	names = append(names, bp.CgoFiles...)

	for _, name := range names {
		blob, readErr := ld.vfs.ReadFile(ld.vfs.Join(dir, name))
		if readErr != nil {
			continue
		}
		if bytes.Contains(blob, []byte(annotationMarker)) {
			return true
		}
	}

	return false
}

// reportExportOnly announces a dependency whose types were read but whose source was not.
func (ld *loadState) reportExportOnly(importPath, why string) {
	if ld.onExportOnly == nil || ld.exportOnlyReported[importPath] {
		return
	}
	ld.exportOnlyReported[importPath] = true

	ld.onExportOnly(ExportOnly{Path: importPath, Reason: why})
}

// importExported returns a package read from export data, completing anything it refers to.
//
// This is the fast path over parsing the file from source, whenever source is not needed immediately by
// codescan to determine the root files that need source parsing.
func (ld *loadState) importExported(importPath string) (*types.Package, error) {
	if pkg, ok := ld.exported[importPath]; ok && pkg.Complete() {
		return pkg, nil
	}
	if ld.exportInProgress[importPath] {
		// Export data has no import cycles, but a corrupt or hand-made tree could; refuse rather than recurse forever.
		return nil, fmt.Errorf("%w through %q in export data", ErrImportCycle, importPath)
	}
	ld.exportInProgress[importPath] = true
	defer delete(ld.exportInProgress, importPath)

	pkg, err := ld.readExported(importPath)
	if err != nil {
		return nil, err
	}

	// Read leaves referenced packages as incomplete placeholders.
	// That is fine until the checker looks inside one — a field whose type lives in another package, say — so complete
	// them eagerly.
	// The closure is only what this package actually refers to, and reading export data is cheap.
	for _, dep := range pkg.Imports() {
		if dep.Complete() || ld.exportInProgress[dep.Path()] {
			continue
		}
		if _, err := ld.importExported(dep.Path()); err != nil {
			// A missing dependency degrades that one package rather than failing the import: the referring package is usually
			// still usable for what codescan asks of it.
			continue
		}
	}

	return pkg, nil
}

func (ld *loadState) readExported(importPath string) (*types.Package, error) {
	blob, err := fs.ReadFile(ld.exportFS, path.Join(importPath)+".export")
	if err != nil {
		return nil, fmt.Errorf("no export data for %q: %w", importPath, err)
	}

	// A generated tree holds bare export sections, which gcexportdata.Read takes directly.
	//
	// A whole compiled archive is accepted too, but has to have its section located first — NewReader only understands
	// the archive form and rejects the bare one.
	var in io.Reader = bytes.NewReader(blob)
	if bytes.HasPrefix(blob, []byte("!<arch>")) {
		if in, err = gcexportdata.NewReader(in); err != nil {
			return nil, fmt.Errorf("locating export data for %q: %w", importPath, err)
		}
	}

	pkg, err := gcexportdata.Read(in, ld.fset, ld.exported, importPath)
	if err != nil {
		return nil, fmt.Errorf("decoding export data for %q: %w", importPath, err)
	}

	return pkg, nil
}

// hasExportData reports whether the configured tree can serve this import path.
func (ld *loadState) hasExportData(importPath string) bool {
	if ld.exportFS == nil {
		return false
	}
	if pkg, ok := ld.exported[importPath]; ok && pkg.Complete() {
		return true
	}

	f, err := ld.exportFS.Open(path.Join(importPath) + ".export")
	if err != nil {
		return false
	}
	_ = f.Close()

	return true
}
