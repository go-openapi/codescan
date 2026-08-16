// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package corpus resolves the trees the benchmarks measure.
//
// The corpora ship with the repository as one archive
// (internal/benchmarks/testdata/corpus.tgz) and are unpacked on demand, so a
// measurement needs a clone and nothing else — no external checkout, no
// environment variable naming somebody's home directory, and no network. Both
// trees carry their dependencies in vendor/, which is what makes that last one
// true.
package corpus

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Archive layout. The archive holds one directory per corpus at its root, and
// unpacking it is what produces Dir for each of them.
const (
	archiveName = "corpus.tgz"
	unpackedDir = "corpus"
	stampName   = ".stamp"

	dirMode  = 0o750
	fileMode = 0o600
)

// The three ways this package refuses to produce a corpus.
var (
	// errUnsafePath guards the extraction against an archive entry that would
	// write outside the destination. Ours never does; an archive is still
	// untrusted input and archive/tar hands over whatever name it was given.
	errUnsafePath = errors.New("archive entry escapes the destination directory")

	errNoSuchCorpus = errors.New("no such corpus")
	errNoCaller     = errors.New("cannot locate the corpus archive: no caller information")
)

// Corpus is one tree to measure.
//
// Two shapes on purpose, because a conclusion that only holds on one of them is
// not a conclusion:
//
//   - dockerctl — a generated CLIENT for a reasonably large API. Light code,
//     198 definitions, no paths: the annotations are all on models.
//   - kubeapi — a generated SERVER for a very large API. Heavy code, four times
//     the source, 222 definitions and 260 paths, so the route and operation
//     rendering paths carry real weight.
type Corpus struct {
	Name    string
	Dir     string // absolute path to the unpacked tree
	Pattern string // package pattern to scan
}

// names is the whole corpus table, in reporting order. Adding a third tree is
// adding a row here and a directory to the archive.
func names() []string { return []string{"dockerctl", "kubeapi"} }

// Ensure unpacks the archive if needed and returns the corpora.
//
// The unpacked tree is stamped with the archive's digest, so a re-rolled archive
// is noticed and unpacked again rather than measured stale — the failure mode
// that would otherwise be invisible, since a stale tree scans perfectly well.
func Ensure() ([]Corpus, error) {
	archive, err := archivePath()
	if err != nil {
		return nil, err
	}

	digest, err := fileDigest(archive)
	if err != nil {
		return nil, err
	}

	dest := filepath.Join(filepath.Dir(archive), unpackedDir)
	if err := unpackIfStale(archive, dest, digest); err != nil {
		return nil, err
	}

	all := names()
	corpora := make([]Corpus, 0, len(all))

	for _, name := range all {
		dir := filepath.Join(dest, name)
		if _, err := os.Stat(dir); err != nil {
			return nil, fmt.Errorf("corpus %q missing from %s: %w", name, archiveName, err)
		}

		corpora = append(corpora, Corpus{Name: name, Dir: dir, Pattern: "./..."})
	}

	return corpora, nil
}

// Find returns one corpus by name.
//
// The name is checked before anything is unpacked. Otherwise a typo pays for the
// whole archive before being told it was a typo — and, less obviously, the test
// covering that refusal would unpack 55 MB on every CI job of every platform,
// which is exactly what the measurement gate exists to prevent.
func Find(name string) (Corpus, error) {
	if !slices.Contains(names(), name) {
		return Corpus{}, fmt.Errorf("%w: %q (have %s)", errNoSuchCorpus, name, strings.Join(names(), ", "))
	}

	corpora, err := Ensure()
	if err != nil {
		return Corpus{}, err
	}

	for _, c := range corpora {
		if c.Name == name {
			return c, nil
		}
	}

	return Corpus{}, fmt.Errorf("%w: %q missing from %s", errNoSuchCorpus, name, archiveName)
}

// archivePath locates the archive from this file's own position in the tree.
//
// The alternative — resolving against the working directory — would mean one
// answer under `go test` (the package directory) and another under `go run`
// (wherever the operator stands).
func archivePath() (string, error) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		return "", errNoCaller
	}

	// this file is internal/benchmarks/corpus/corpus.go; the archive is a
	// sibling of the package directory, under testdata.
	path := filepath.Join(filepath.Dir(filepath.Dir(self)), "testdata", archiveName)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("corpus archive: %w", err)
	}

	return path, nil
}

// unpackIfStale unpacks the archive unless the destination already carries a
// stamp matching its digest.
func unpackIfStale(archive, dest, digest string) error {
	stamp := filepath.Join(dest, stampName)
	if got, err := os.ReadFile(stamp); err == nil && string(got) == digest {
		return nil
	}

	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("clearing %s: %w", dest, err)
	}

	if err := unpack(archive, dest); err != nil {
		return err
	}

	return os.WriteFile(stamp, []byte(digest), fileMode)
}

// unpack extracts a gzipped tar into dest.
func unpack(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}

	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("reading %s: %w", archive, err)
	}

	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("reading %s: %w", archive, err)
		}

		if err := extractEntry(tr, hdr, dest); err != nil {
			return err
		}
	}
}

// extractEntry writes one archive entry. Only directories and regular files are
// honoured: the corpora are generated Go source and vendored dependencies, so
// anything else is not something to reproduce silently.
func extractEntry(tr *tar.Reader, hdr *tar.Header, dest string) error {
	target, err := safeJoin(dest, hdr.Name)
	if err != nil {
		return err
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(target, dirMode)
	case tar.TypeReg:
		if err := os.MkdirAll(filepath.Dir(target), dirMode); err != nil {
			return err
		}

		return writeFile(tr, target, hdr.Size)
	default:
		return nil
	}
}

// writeFile copies exactly size bytes out of the archive.
//
// The bound is the point: io.Copy over an untrusted archive is a decompression
// bomb waiting to happen, and the header already says how long the entry is.
func writeFile(r io.Reader, target string, size int64) error {
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return err
	}

	if _, err := io.CopyN(out, r, size); err != nil && !errors.Is(err, io.EOF) {
		_ = out.Close()

		return fmt.Errorf("writing %s: %w", target, err)
	}

	return out.Close()
}

// safeJoin resolves an archive entry name against dest, refusing anything that
// is not a plain relative path inside it.
//
// The name is judged in the SLASH form the tar format defines, never in the
// host's. A host-shaped check accepts on one platform what it refuses on another:
// filepath.IsAbs("/x") is false on Windows, which needs a volume — so the entry
// that Linux refused as absolute was silently accepted there. Backslash and colon
// go the same way, being a separator and a drive marker on Windows and ordinary
// filename characters here, so both are refused everywhere rather than in one
// place.
//
// Refusing rather than clamping: the usual sanitisation rewrites `../x` to `x`
// and carries on, which for an archive this repository rolls itself would turn a
// broken corpus into a quietly misplaced one.
func safeJoin(dest, name string) (string, error) {
	if name == "" || strings.ContainsAny(name, `\:`) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("%w: %q", errUnsafePath, name)
	}

	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%w: %q", errUnsafePath, name)
	}

	target := filepath.Join(dest, filepath.FromSlash(clean))

	// The same refusal restated on the joined path. Redundant after the checks
	// above, and worth keeping: it is the containment property this function
	// exists to guarantee, asserted on the value it actually returns.
	if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %q", errUnsafePath, name)
	}

	return target, nil
}

// fileDigest returns the SHA-256 of a file, hex-encoded.
func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("digesting %s: %w", path, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
