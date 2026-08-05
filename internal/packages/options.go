// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package packages

import (
	"go/token"
	"io/fs"
)

// Option configures a [Loader].
type Option func(*options)

type options struct {
	strategy     Strategy // zero value: StrategyGoPackages
	fsys         fs.FS    // nil: read through the real filesystem
	goEnv        GoEnv
	stubStdlib   bool
	onSynthesize func(Synthesized)
	exportFS     fs.FS
}

// WithExportData serves dependencies from pre-computed export data instead of reading their source.
//
// fsys holds one file per package, named by import path with a ".export" suffix — the layout
// hack/genexportdata produces. Whole compiled archives are accepted as well as bare export sections.
//
// It applies to dependencies only. The module under scan is always read from source: its comments are
// the annotations, and export data carries none.
//
// This is the fast path with none of the fidelity loss of [WithStubbedStdlib]: the types are the ones
// the compiler computed, so fields, method sets and interface identity are all real. A full scan
// otherwise spends nearly all of its time parsing and type-checking its dependencies, and a
// WebAssembly guest pays a five- to six-fold compute tax on top of that.
//
// The data is only valid for the toolchain that produced it, since the export format is tied to the
// Go release. A package the tree does not cover falls back to source, and then to synthesis.
func WithExportData(fsys fs.FS) Option {
	return func(o *options) { o.exportFS = fsys }
}

// Synthesized reports an import whose types were fabricated rather than loaded.
type Synthesized struct {
	// Path is the import path.
	Path string

	// Pos is the import that triggered the synthesis — the first one seen for this path.
	Pos token.Position

	// Deliberate distinguishes the standard library withheld by [WithStubbedStdlib] from an import
	// that simply could not be found. The first is the caller's own choice; the second is usually a
	// mounting or module-cache problem.
	Deliberate bool

	// Cgo marks the "C" pseudo-package, which is neither of the above: it has no source anywhere to
	// find, and is fabricated because this loader does not run the cgo tool. Worth telling apart, since
	// "could not be resolved" invites a reader to go looking for something that was never there.
	Cgo bool
}

// WithOnSynthesized registers a callback fired once per import path that had to be synthesized.
//
// Without it, the loss is invisible: a package that only mentions a synthesized type in a field
// position type-checks cleanly and simply produces a thinner spec. What surfaces otherwise is the
// downstream wreckage — a value-position use of a fabricated type reads as an error in the scanned
// code rather than as a missing dependency.
func WithOnSynthesized(fn func(Synthesized)) Option {
	return func(o *options) { o.onSynthesize = fn }
}

// WithStubbedStdlib keeps the standard library out of the package graph.
//
// Standard-library imports are then synthesized from the names selected through them — opaque types
// carrying the right package path and name — instead of being parsed and type-checked out of GOROOT.
//
// This trades fidelity for reach. What survives is everything keyed on a type's identity: codescan
// recognizes time.Time, json.RawMessage and friends by (package, name), never by shape. What is lost
// is everything structural: a synthesized type has no fields to drill into and no method set, so a
// spec that renders json.RawMessage as []byte, or that depends on a type implementing
// encoding.TextMarshaler, will come out different.
//
// The reach it buys is real: GOROOT no longer has to exist, which for a WASI guest or a browser means
// there is no standard-library source tree to ship or mount.
//
// # Not failsafe
//
// This mode trades correctness guarantees for reach, and the trade is not always visible in the
// output. What it buys: a small footprint, no Go installation, and no module cache to populate — the
// scan reads only the project tree. What it costs is that a synthesized type has no structure, so a
// spec can come out subtly thinner rather than failing loudly. Across codescan's own fixture corpus
// 133 of 138 scans are byte-identical; the rest lose a byte-array rendering, an integer format, or a
// TextMarshaler-derived string, and stdlib interfaces such as io.Reader have no identity recognizer
// to fall back on at all.
//
// Note that synthesis is not exclusive to this option: an import that cannot be resolved is
// synthesized whether or not the standard library was withheld. The option only makes it deliberate
// for the one dependency every Go program has.
//
// Prefer a full graph wherever GOROOT is available; reach for this where it is not.
func WithStubbedStdlib() Option {
	return func(o *options) { o.stubStdlib = true }
}

// WithFS makes the loader read source through fsys instead of the real filesystem.
//
// Every path the loader is given — Config.Dir, the patterns, and the paths it derives from them — is
// then interpreted relative to the root of fsys, following [io/fs] conventions: slash-separated, no
// leading slash, no "..". A leading slash or an OS-specific separator is normalised away rather than
// rejected, so a caller can pass the same patterns it would use natively.
//
// This is the seam that makes a virtualized source tree possible: an in-memory tree in a WASI guest,
// a [testing/fstest.MapFS] in a unit test, an archive reader, or an overlay composed from several
// roots. The default (no WithFS) reads through the os package.
//
// It also forces [StrategyToolchainFree], overriding [WithStrategy]: `go list` runs against the real
// filesystem, so it could not honour fsys even if asked.
func WithFS(fsys fs.FS) Option {
	return func(o *options) { o.fsys = fsys }
}

func newOptions(opts []Option) *options {
	o := &options{}
	for _, apply := range opts {
		apply(o)
	}
	return o
}
