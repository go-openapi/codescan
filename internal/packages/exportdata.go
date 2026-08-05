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
// 190 packages and 1195 files for a fixture as small as the petstore, and a WebAssembly guest pays a
// five- to six-fold compute tax on top. None of that work is discovery — the answers were already
// computed when the toolchain built those packages, and the compiler wrote them down.
//
// So read them instead. The saving is in the parsing and type-checking avoided, not in the I/O:
// filesystem syscalls account for under 2% of a full WASI scan.

// importExported returns a package read from export data, completing anything it refers to.
func (ld *loadState) importExported(importPath string) (*types.Package, error) {
	if pkg, ok := ld.exported[importPath]; ok && pkg.Complete() {
		return pkg, nil
	}
	if ld.exportInProgress[importPath] {
		// Export data has no import cycles, but a corrupt or hand-made tree could; refuse rather than
		// recurse forever.
		return nil, fmt.Errorf("%w through %q in export data", ErrImportCycle, importPath)
	}
	ld.exportInProgress[importPath] = true
	defer delete(ld.exportInProgress, importPath)

	pkg, err := ld.readExported(importPath)
	if err != nil {
		return nil, err
	}

	// Read leaves referenced packages as incomplete placeholders. That is fine until the checker looks
	// inside one — a field whose type lives in another package, say — so complete them eagerly. The
	// closure is only what this package actually refers to, and reading export data is cheap.
	for _, dep := range pkg.Imports() {
		if dep.Complete() || ld.exportInProgress[dep.Path()] {
			continue
		}
		if _, err := ld.importExported(dep.Path()); err != nil {
			// A missing dependency degrades that one package rather than failing the import: the
			// referring package is usually still usable for what codescan asks of it.
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

	// A generated tree holds bare export sections, which gcexportdata.Read takes directly. A whole
	// compiled archive is accepted too, but has to have its section located first — NewReader only
	// understands the archive form and rejects the bare one.
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
