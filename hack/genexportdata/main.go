// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Command genexportdata extracts the compiler's export data for a set of packages into a directory
// tree keyed by import path.
//
// The result is what internal/packages reads through WithExportData (codescan.Options.ExportData):
// pre-digested types, so a scan never has to parse or type-check those packages. It is generated
// here, natively, because producing it needs the toolchain — which is precisely what the consumer
// does not have.
//
//	go run ./hack/genexportdata -out ./exportdata std
//	go run ./hack/genexportdata -dir hack/genexportdata/bundle -out bundle.zip std ./...
//
// The bundle module names what the published archive covers; see hack/genexportdata/bundle.
//
// The output is valid only for the toolchain that produced it. Export data format is tied to the Go
// release, so regenerate it whenever the supported toolchain moves.
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/gcexportdata"
)

// Sizes and permissions, named so they read as intent rather than as numbers.
const (
	bytesPerMB = 1 << 20
	dirPerm    = 0o750
)

func main() {
	out := flag.String("out", "exportdata", "directory, or .zip file, to write the export data into")
	dir := flag.String("dir", "", "module to resolve the patterns in (default: the current one)")
	withAnnotated := flag.Bool("with-annotated", false,
		"include packages carrying swagger annotations (only safe when their source ships too)")
	flag.Parse()

	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"std"}
	}
	moduleDir = *dir
	includeAnnotated = *withAnnotated

	write := run
	if strings.HasSuffix(*out, ".zip") {
		// A zip is what an embedded build wants: archive/zip's Reader is already an fs.FS, so the
		// artifact carries one file and still serves per-package reads lazily.
		write = runZip
	}

	n, bytes, err := write(*out, patterns)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genexportdata:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d packages, %.1f MB, to %s\n", n, float64(bytes)/bytesPerMB, *out)
}

// runZip writes the same tree into a single archive.
func runZip(dest string, patterns []string) (int, int64, error) {
	pkgs, err := list(patterns)
	if err != nil {
		return 0, 0, err
	}

	if err := os.MkdirAll(filepath.Dir(dest), dirPerm); err != nil {
		return 0, 0, err
	}
	f, err := os.Create(dest)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	var count int
	for _, p := range pkgs {
		if p.Export == "" {
			continue
		}
		if skipAnnotated(p) {
			continue
		}
		w, err := zw.Create(p.ImportPath + ".export")
		if err != nil {
			return 0, 0, err
		}
		if err := copyExportSection(p.Export, w); err != nil {
			return 0, 0, fmt.Errorf("%s: %w", p.ImportPath, err)
		}
		count++
	}
	if err := zw.Close(); err != nil {
		return 0, 0, err
	}

	info, err := f.Stat()
	if err != nil {
		return 0, 0, err
	}

	return count, info.Size(), nil
}

// moduleDir is where `go list` runs, so a bundle can be resolved in a module that requires what it
// should cover rather than in whichever one the tool was invoked from.
var moduleDir string //nolint:gochecknoglobals // set once from the command line

// includeAnnotated keeps annotated packages in the output. See skipAnnotated.
var includeAnnotated bool //nolint:gochecknoglobals // set once from the command line

// listed is the slice of `go list -json` output this tool needs.
//
// The field names are the go command's, not ours: these tags describe someone else's JSON.
//
//nolint:tagliatelle // go list emits PascalCase keys
type listed struct {
	ImportPath string   `json:"ImportPath"`
	Export     string   `json:"Export"`
	Standard   bool     `json:"Standard"`
	Dir        string   `json:"Dir"`
	GoFiles    []string `json:"GoFiles"`
}

// annotated reports whether a package carries swagger annotations in its comments.
//
// Such a package must never go into the bundle. Export data holds types, not comments, so a package
// whose meaning is written in annotations — strfmt declaring its formats with `swagger:strfmt`, say —
// comes back structurally intact and semantically empty. Nothing errors; the spec is just quietly
// poorer. The scan has to read those from source.
func annotated(p listed) bool {
	for _, name := range p.GoFiles {
		blob, err := os.ReadFile(filepath.Join(p.Dir, name))
		if err != nil {
			continue
		}
		if bytes.Contains(blob, []byte("swagger:")) {
			return true
		}
	}

	return false
}

func run(outDir string, patterns []string) (int, int64, error) {
	pkgs, err := list(patterns)
	if err != nil {
		return 0, 0, err
	}

	var count int
	var total int64
	for _, p := range pkgs {
		if p.Export == "" {
			continue // no compiled form: a package with no Go files, or one that failed to build
		}
		if skipAnnotated(p) {
			continue
		}
		written, err := extract(p.Export, filepath.Join(outDir, filepath.FromSlash(p.ImportPath)+".export"))
		if err != nil {
			return 0, 0, fmt.Errorf("%s: %w", p.ImportPath, err)
		}
		count++
		total += written
	}

	return count, total, nil
}

// skipAnnotated drops a package that carries annotations, saying so: a silent omission here would
// show up much later as a thinner spec with nothing to point at.
//
// -with-annotated turns this off, and there is now a case for it. The loader reads a dependency's
// COMMENTS from its source even when its types come from export data, so where that source ships
// alongside the bundle an annotated package is safe to include and is simply faster. Where it does
// not — an archive handed to a WebAssembly guest with no source anywhere — the omission is the point.
func skipAnnotated(p listed) bool {
	if includeAnnotated || !annotated(p) {
		return false
	}
	fmt.Fprintf(os.Stderr, "skipping %s: carries swagger annotations, which export data cannot hold — "+
		"it has to be read from source\n", p.ImportPath)

	return true
}

// list asks the go command to build the patterns and report where it put each compiled archive.
func list(patterns []string) ([]listed, error) {
	args := append([]string{"list", "-export", "-json=ImportPath,Export,Standard,Dir,GoFiles", "-deps"}, patterns...)
	cmd := exec.CommandContext(context.Background(), "go", args...)
	cmd.Dir = moduleDir
	if moduleDir != "" {
		// -dir names a module to resolve in, with its own pins. A go.work above it would override
		// exactly that, and refuse outright when the module is not one the workspace uses — which is the
		// normal case for a manifest module that deliberately keeps its versions to itself.
		cmd.Env = append(os.Environ(), "GOWORK=off")
	}

	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err, stderr.String())
	}

	var pkgs []listed
	dec := json.NewDecoder(strings.NewReader(string(stdout)))
	for dec.More() {
		var p listed
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("decoding go list output: %w", err)
		}
		pkgs = append(pkgs, p)
	}

	return pkgs, nil
}

// extract copies just the export section out of a compiled archive.
//
// The archive also holds object code, which is the bulk of it and of no use here: the export section
// alone is around a twentieth of the size.
func extract(archive, dest string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(dest), dirPerm); err != nil {
		return 0, err
	}
	w, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer func() { _ = w.Close() }()

	if err := copyExportSection(archive, w); err != nil {
		return 0, err
	}

	info, err := w.Stat()
	if err != nil {
		return 0, err
	}

	return info.Size(), w.Close()
}

// copyExportSection writes just the export section of a compiled archive to w.
//
// The archive also holds object code, which is the bulk of it and of no use here: the export section
// alone is around a twentieth of the size.
func copyExportSection(archive string, w io.Writer) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	r, err := gcexportdata.NewReader(f)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, r)

	return err
}
