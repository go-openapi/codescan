// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Command genspec-wasi scans annotated Go source and writes the Swagger specification it describes.
//
// It is a headless counterpart to genspec-tui, and the form codescan takes when it is built for
// WebAssembly: it carries no dependency beyond the library itself, so it cross-compiles to
// wasip1/wasm and runs under any WASI runtime.
//
//	go build -o genspec-wasi ./cmd/genspec-wasi
//	GOOS=wasip1 GOARCH=wasm go build -o genspec-wasi.wasm ./cmd/genspec-wasi
//	wazero run -mount /path/to/project:/src genspec-wasi.wasm -workdir /src ./...
//
// Under a WASI runtime the scan reads only what the host mounted, and runs no subprocess: use
// -loader=own, which needs neither the go command nor a toolchain. See -h for the full flag set.
//
// # Mounting, and what it costs
//
// A full scan resolves the standard library and the module cache by path, so both have to be mounted
// and named through GOROOT and GOMODCACHE. -stub-stdlib removes the first requirement, and an
// unresolvable import is synthesized rather than fatal, so a scan will complete with nothing but the
// project tree mounted. Measured on codescan's petstore fixture under wasmtime:
//
//	everything mounted, full graph      8.1 s   848 MB   byte-identical
//	module cache only, -stub-stdlib     1.0 s   143 MB   byte-identical
//	project tree only, -stub-stdlib     0.1 s   123 MB   one format lost
//
// The last row is the shape of the trade: it does not fail, it quietly emits slightly less. Prefer a
// full graph wherever GOROOT is available.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/exportdata"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "genspec-wasi:", err)
		os.Exit(1)
	}
}

type config struct {
	set *flag.FlagSet

	workdir    *string
	bools      map[string]*bool
	buildTags  *string
	goos       *string
	goarch     *string
	loader     *string
	exportData *string
	format     *string
	output     *string
	indent     *bool
	quiet      *bool
}

func registerFlags(fs *flag.FlagSet) *config {
	return &config{
		set:       fs,
		workdir:   fs.String("workdir", ".", "directory the scan runs from; patterns are relative to it"),
		bools:     registerBools(fs),
		buildTags: fs.String("build-tags", "", "comma-separated go build tags to apply while loading"),
		goos:      fs.String("goos", "", "GOOS the scanned code is built for (default: this machine's)"),
		goarch:    fs.String("goarch", "", "GOARCH the scanned code is built for (default: this machine's)"),
		loader: fs.String("loader", "auto",
			`package loader: "go" runs go list, "own" needs no toolchain, "auto" picks own where there is no exec`),
		exportData: fs.String("export-data", "",
			"directory or .zip of pre-computed export data for dependencies (see hack/genexportdata):\n"+
				"full fidelity, none of the cost of type-checking them from source"),
		format: fs.String("format", "spec",
			`what to write: "spec" is the document alone, "json" wraps it with the scan's diagnostics and`+"\n"+
				`provenance, positioned and machine-readable (-quiet is moot: nothing goes to stderr)`),
		output: fs.String("output", "-", `write the specification here ("-" for standard output)`),
		indent: fs.Bool("indent", true, "indent the emitted JSON"),
		quiet:  fs.Bool("quiet", false, "suppress scan diagnostics on standard error"),
	}
}

func run(argv []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("genspec-wasi", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: genspec-wasi [flags] [packages...]")
		fmt.Fprintln(stderr, "\nScans annotated Go source and writes a Swagger 2.0 specification.")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}

	cfg := registerFlags(fs)
	if err := fs.Parse(argv); err != nil {
		return err
	}

	wrapped, err := resolveFormat(*cfg.format)
	if err != nil {
		return err
	}

	var sink *collector
	if wrapped {
		sink = &collector{}
	}

	opts, err := cfg.options(fs.Args(), stderr, sink)
	if err != nil {
		return err
	}

	doc, err := codescan.Run(opts)
	if err != nil {
		return err
	}

	return cfg.emit(doc, sink, stdout)
}

// resolveFormat reports whether the document is wrapped with what the scan observed.
//
// Shaped like resolveLoader below: the flag is answered once, here, and everything downstream
// switches on a collector being present rather than on a format string threaded to each site.
func resolveFormat(format string) (bool, error) {
	switch format {
	case "spec":
		return false, nil
	case "json":
		return true, nil
	default:
		return false, fmt.Errorf("%w: -format %q is not one of spec, json", errBadFlag, format)
	}
}

func (c *config) options(patterns []string, stderr io.Writer, sink *collector) (*codescan.Options, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	opts := &codescan.Options{
		Packages:  patterns,
		WorkDir:   *c.workdir,
		BuildTags: *c.buildTags,
		GOOS:      *c.goos,
		GOARCH:    *c.goarch,
	}
	applyBools(opts, c.bools)

	switch {
	case *c.exportData != "":
		data, err := openExportData(*c.exportData)
		if err != nil {
			return nil, err
		}
		opts.ExportData = data
	default:
		// A build carrying its own copy needs nothing mounted for its dependencies.
		if embedded, ok := exportdata.Embedded(); ok {
			opts.ExportData = embedded
		}
	}

	useOwn, err := resolveLoader(*c.loader)
	if err != nil {
		return nil, err
	}
	opts.ToolchainFreeLoader = useOwn
	if useOwn {
		// The loader reads the host filesystem directly, so -workdir has to name a place that exists
		// there rather than a place relative to wherever the process happens to be. Under WASI it never
		// is: a guest starts at "/", which is not a preopen, so only an absolute path names a mount.
		abs, err := absolutePath(*c.workdir)
		if err != nil {
			return nil, err
		}
		opts.WorkDir = abs
	}

	switch {
	case sink != nil:
		// The scan reports positions in the tree it read, which is opts.WorkDir and not necessarily
		// where the caller thinks its files are. Tell the collector before anything is reported.
		sink.root = opts.WorkDir
		opts.OnDiagnostic = sink.onDiagnostic
		opts.OnProvenance = sink.onProvenance
	case !*c.quiet:
		opts.OnDiagnostic = func(d codescan.Diagnostic) {
			fmt.Fprintln(stderr, d.String())
		}
	}

	return opts, nil
}

// resolveLoader reports whether to use codescan's own loader.
//
// "auto" asks whether this build can start a subprocess at all: on wasm it cannot, so `go list` is
// not an option and the choice makes itself.
func resolveLoader(mode string) (bool, error) {
	switch mode {
	case "own":
		return true, nil
	case "go":
		return false, nil
	case "auto":
		return !canExec(), nil
	default:
		return false, fmt.Errorf("%w: -loader %q is not one of go, own, auto", errBadFlag, mode)
	}
}

// absolutePath resolves p against the working directory, in the host's own notion of a path.
//
// Absolute means what the host means by it, not "starts with a slash": a Windows -workdir starts
// with a drive letter, so reading it as relative appends it to the working directory and yields a
// path that names nothing. Under WASI the two notions coincide, and the working directory is
// consulted only for a relative -workdir -- an absolute one names the mount and returns before
// os.Getwd, which a guest need not have.
func absolutePath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("cannot resolve %q against the working directory: %w", p, err)
	}

	return abs, nil
}

func (c *config) emit(doc any, sink *collector, stdout io.Writer) error {
	out := stdout
	if *c.output != "-" {
		f, err := os.Create(*c.output)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		out = f
	}

	enc := json.NewEncoder(out)
	if *c.indent {
		enc.SetIndent("", "  ")
	}

	if sink == nil {
		return enc.Encode(doc)
	}

	return enc.Encode(sink.result(doc))
}
