// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// allowedWASIImports is everything the artifact may ask a host to provide.
//
// Every entry reads. That is not an accident of the current dependencies but a property worth
// keeping: it is what lets a host refuse writes outright, and it shrinks what a browser shim has to
// implement correctly before the scanner will run at all.
//
// The list was 24 until the go/packages strategy was refused on WebAssembly builds -
// path_create_directory, path_remove_directory and path_unlink_file arrived through the temporary
// files `go list` needs, not through anything a scan does. The refusal is a build-tagged call site
// (internal/packages/golist_wasm.go), so the driver is unreachable and the linker drops it, even
// though the package itself stays imported for its type aliases. Linking one package that writes
// would put those three back silently.
var allowedWASIImports = map[string]string{ //nolint:gochecknoglobals // table for the guard below
	"args_get":            "argv",
	"args_sizes_get":      "argv",
	"clock_time_get":      "the runtime's timers",
	"environ_get":         "GOROOT, GOMODCACHE and the build target",
	"environ_sizes_get":   "as above",
	"fd_close":            "reading source",
	"fd_fdstat_get":       "reading source",
	"fd_fdstat_set_flags": "reading source",
	"fd_filestat_get":     "reading source",
	"fd_prestat_dir_name": "discovering what the host mounted",
	"fd_prestat_get":      "discovering what the host mounted",
	"fd_pread":            "reading an export-data archive at an offset, without seeking",
	"fd_read":             "reading source",
	"fd_readdir":          "walking a package directory",
	"fd_write":            "the specification, on stdout, and diagnostics on stderr",
	"path_filestat_get":   "resolving a package directory",
	"path_open":           "reading source",
	"path_readlink":       "resolving a symlinked directory",
	"poll_oneoff":         "the Go scheduler's timers",
	"proc_exit":           "termination",
	"random_get":          "map seeding",
	"sched_yield":         "the Go scheduler",
}

// TestWASIArtifactImportsOnlyReads pins what the artifact demands of a host.
//
// It needs a Go toolchain to build the artifact but no WASI runtime to inspect it, so it is the one
// check in this file that can run anywhere.
func TestWASIArtifactImportsOnlyReads(t *testing.T) {
	if testing.Short() {
		t.Skip("cross-compiles the artifact")
	}
	t.Parallel()

	artifact := buildWASIArtifact(t)

	blob, err := os.ReadFile(artifact)
	require.NoError(t, err)

	imports, err := wasmImports(blob)
	require.NoError(t, err)
	require.NotEmpty(t, imports, "no imports found; the parser is probably wrong, not the artifact")

	// A module may import the same function more than once; report each name once.
	unexpectedSet := map[string]struct{}{}
	for _, imp := range imports {
		// A second import module would mean the artifact stopped being plain WASI - GOOS=js, for
		// instance, needs its own `go` module and a JavaScript runtime to go with it.
		assert.Equal(t, "wasi_snapshot_preview1", imp.module,
			"import %q comes from %q: the artifact is no longer hostable by a plain WASI runtime", imp.name, imp.module)

		// Anything that is not a function is a shared resource - an imported memory in particular would
		// mean SharedArrayBuffer, and with it the COOP/COEP headers that rule out plain static hosting.
		assert.Equal(t, wasmImportFunc, imp.kind,
			"import %q is not a function: the artifact now shares state with its host", imp.name)

		if _, ok := allowedWASIImports[imp.name]; !ok {
			unexpectedSet[imp.name] = struct{}{}
		}
	}

	unexpected := make([]string, 0, len(unexpectedSet))
	for name := range unexpectedSet {
		unexpected = append(unexpected, name)
	}
	sort.Strings(unexpected)

	assert.Empty(t, unexpected,
		"the artifact asks its host for %v.\n"+
			"Something now linked into the WebAssembly build needs more than reading source. Either drop it from "+
			"the wasm build (see internal/packages/golist_wasm.go for how the go/packages strategy is refused), or "+
			"add it here with the reason, knowing every browser host must then implement it.", unexpected)
}

const (
	wasmImportFunc byte = iota
	wasmImportTable
	wasmImportMemory
	wasmImportGlobal
)

// Failures the decoder can meet. Sentinels rather than a formatted string each time: a test that
// asserts on a malformed module should be able to say which malformation it expects.
var (
	errNotWasm     = errors.New("not a WebAssembly module")
	errTruncated   = errors.New("truncated module")
	errBadUvarint  = errors.New("malformed integer")
	errUnknownKind = errors.New("unknown import kind")
	// errTooLarge guards the conversions below. A length read out of the file is a uint64, and int is
	// 32 bits on some targets, so a corrupt or hostile module could otherwise wrap one into a negative
	// index. The module here is one we built, which makes this cheap insurance rather than a live
	// concern - but it is the sort of insurance that stops being cheap once it is missing.
	errTooLarge = errors.New("length beyond what this platform can index")
)

// atMost converts a length read from the module, refusing one this platform cannot hold.
//
// The bound is checked in the uint64 domain and limit is a slice length, so it is never negative and
// the widening is exact. Once n is known to be within it, narrowing to int cannot wrap.
func atMost(n uint64, limit int) (int, error) {
	if limit < 0 {
		return 0, fmt.Errorf("%w: negative bound %d", errTooLarge, limit)
	}
	if n > uint64(limit) {
		return 0, fmt.Errorf("%w: %d", errTooLarge, n)
	}

	// Provably in range by the two checks above - limit is non-negative so the widening is exact, and
	// n is no larger than it, hence no larger than MaxInt. The analyser does not carry that across the
	// return, and restructuring the guards to suit it only obscured why they are there.
	return int(n), nil //nolint:gosec // bounded immediately above
}

type wasmImport struct {
	module string
	name   string
	kind   byte
}

// wasmImports decodes the import section of a WebAssembly module.
//
// Hand-rolled because the alternative is a dependency on a wasm toolchain that only this guard would
// use; the encoding is a handful of LEB128 integers and length-prefixed strings.
func wasmImports(blob []byte) ([]wasmImport, error) {
	if len(blob) < 8 || string(blob[:4]) != "\x00asm" {
		return nil, errNotWasm
	}

	r := &wasmReader{blob: blob, pos: 8}
	for r.pos < len(blob) {
		id, err := r.byte()
		if err != nil {
			return nil, err
		}
		size, err := r.uvarint()
		if err != nil {
			return nil, err
		}
		if id != 2 { // 2 is the import section
			skip, err := atMost(size, len(blob)-r.pos)
			if err != nil {
				return nil, err
			}
			r.pos += skip

			continue
		}

		return r.importSection()
	}

	return nil, nil
}

type wasmReader struct {
	blob []byte
	pos  int
}

func (r *wasmReader) byte() (byte, error) {
	if r.pos >= len(r.blob) {
		return 0, fmt.Errorf("%w: at %d", errTruncated, r.pos)
	}
	b := r.blob[r.pos]
	r.pos++

	return b, nil
}

func (r *wasmReader) uvarint() (uint64, error) {
	v, n := binary.Uvarint(r.blob[r.pos:])
	if n <= 0 {
		return 0, fmt.Errorf("%w: at %d", errBadUvarint, r.pos)
	}
	r.pos += n

	return v, nil
}

func (r *wasmReader) name() (string, error) {
	n, err := r.uvarint()
	if err != nil {
		return "", err
	}
	length, err := atMost(n, len(r.blob)-r.pos)
	if err != nil {
		return "", fmt.Errorf("%w: at %d", errTruncated, r.pos)
	}
	s := string(r.blob[r.pos : r.pos+length])
	r.pos += length

	return s, nil
}

// limits skips a limits record, which follows every table and memory import.
func (r *wasmReader) limits() error {
	flag, err := r.byte()
	if err != nil {
		return err
	}
	if _, err := r.uvarint(); err != nil { // minimum
		return err
	}
	if flag&0x01 == 0 {
		return nil
	}
	_, err = r.uvarint() // maximum

	return err
}

func (r *wasmReader) importSection() ([]wasmImport, error) {
	count, err := r.uvarint()
	if err != nil {
		return nil, err
	}

	imports := make([]wasmImport, 0, count)
	for range count {
		module, err := r.name()
		if err != nil {
			return nil, err
		}
		name, err := r.name()
		if err != nil {
			return nil, err
		}
		kind, err := r.byte()
		if err != nil {
			return nil, err
		}

		switch kind {
		case wasmImportFunc:
			_, err = r.uvarint() // type index
		case wasmImportTable:
			if _, err = r.byte(); err == nil { // element type
				err = r.limits()
			}
		case wasmImportMemory:
			err = r.limits()
		case wasmImportGlobal:
			if _, err = r.byte(); err == nil { // value type
				_, err = r.byte() // mutability
			}
		default:
			err = fmt.Errorf("%w: %d for %q", errUnknownKind, kind, name)
		}
		if err != nil {
			return nil, err
		}

		imports = append(imports, wasmImport{module: module, name: name, kind: kind})
	}

	return imports, nil
}
