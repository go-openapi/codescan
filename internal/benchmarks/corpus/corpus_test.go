// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package corpus

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// envBenchEnabled gates the test that unpacks the real corpus — 5000 files and 56 MB, which is not
// something to write on every CI job of every platform. The extraction itself is covered below
// against a synthetic archive, which exercises the same code for a few hundred bytes.
const envBenchEnabled = "CODESCAN_BENCH"

// TestEnsure covers the property the benchmarks depend on: after Ensure, every corpus is a real
// module tree on disk, and asking again is cheap rather than a second extraction.
func TestEnsure(t *testing.T) {
	if os.Getenv(envBenchEnabled) != "1" {
		t.Skipf("%s!=1: skipping the unpacking of the real corpus", envBenchEnabled)
	}

	corpora, err := Ensure()
	require.NoError(t, err)
	require.Len(t, corpora, len(names()))

	for _, c := range corpora {
		t.Run(c.Name, func(t *testing.T) {
			assert.Equal(t, "./...", c.Pattern)
			assert.True(t, filepath.IsAbs(c.Dir), "a corpus directory is resolved, not relative")

			// A go.mod, because every measurement scans a module; a vendor directory, because that
			// is what makes the corpus readable with no network.
			assert.FileExists(t, filepath.Join(c.Dir, "go.mod"))
			assert.DirExists(t, filepath.Join(c.Dir, "vendor"))
		})
	}

	t.Run("is idempotent", func(t *testing.T) {
		again, err := Ensure()
		require.NoError(t, err)
		assert.Equal(t, corpora, again)
	})

	t.Run("finds one by name", func(t *testing.T) {
		c, err := Find(names()[0])
		require.NoError(t, err)
		assert.Equal(t, names()[0], c.Name)
	})
}

// TestUnpackIfStale covers the extraction against a synthetic archive: what it writes, that it
// skips a destination whose stamp already matches, and that it starts over when it does not.
//
// The staleness half is the one that matters. A re-rolled archive measured against the tree the
// previous one left behind would report perfectly plausible figures for the wrong corpus.
func TestUnpackIfStale(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "synthetic.tgz")
	dest := filepath.Join(dir, "unpacked")

	writeArchive(t, archive, map[string]string{
		"acorpus/go.mod":          "module example.com/acorpus\n",
		"acorpus/vendor/dep/x.go": "package dep\n",
	})

	digest, err := fileDigest(archive)
	require.NoError(t, err)

	require.NoError(t, unpackIfStale(archive, dest, digest))
	assert.FileExists(t, filepath.Join(dest, "acorpus", "go.mod"))
	assert.FileExists(t, filepath.Join(dest, "acorpus", "vendor", "dep", "x.go"))

	t.Run("leaves a matching destination alone", func(t *testing.T) {
		// A file the archive does not carry survives only if nothing was re-extracted.
		witness := filepath.Join(dest, "witness")
		require.NoError(t, os.WriteFile(witness, []byte("kept"), fileMode))

		require.NoError(t, unpackIfStale(archive, dest, digest))
		assert.FileExists(t, witness)
	})

	t.Run("starts over when the stamp differs", func(t *testing.T) {
		require.NoError(t, unpackIfStale(archive, dest, "a different digest"))
		assertAbsent(t, filepath.Join(dest, "witness"))
		assert.FileExists(t, filepath.Join(dest, "acorpus", "go.mod"))

		stamp, err := os.ReadFile(filepath.Join(dest, stampName))
		require.NoError(t, err)
		assert.Equal(t, "a different digest", string(stamp))
	})
}

// TestUnpackRefusesAnEscapingEntry pins the refusal on a whole archive rather than on safeJoin
// alone: an entry that escapes must fail the extraction, not be quietly relocated inside it.
func TestUnpackRefusesAnEscapingEntry(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "hostile.tgz")

	writeArchive(t, archive, map[string]string{"../escaped.go": "package escaped\n"})

	err := unpack(archive, filepath.Join(dir, "unpacked"))
	require.ErrorIs(t, err, errUnsafePath)
	assertAbsent(t, filepath.Join(dir, "escaped.go"))
}

// TestFindRejectsAnUnknownName covers what an operator sees after a typo.
func TestFindRejectsAnUnknownName(t *testing.T) {
	_, err := Find("nosuchcorpus")
	require.ErrorIs(t, err, errNoSuchCorpus)
	assert.Contains(t, err.Error(), names()[0], "the error names what there is to choose from")
}

// TestSafeJoin pins the path rule directly.
//
// The rule has to be the SAME on every platform, which is what the last three cases are about: a
// leading `/` is not "absolute" to Windows, and `\` and `:` are separators and drive markers there
// while being ordinary filename characters here. Judging the name in the host's terms let Windows
// accept what Linux refused — the divergence CI caught.
func TestSafeJoin(t *testing.T) {
	dest := t.TempDir()

	t.Run("resolves an ordinary entry", func(t *testing.T) {
		got, err := safeJoin(dest, "dockerctl/go.mod")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dest, "dockerctl", "go.mod"), got)
	})

	for name, why := range map[string]string{
		"../escaped.go":              "climbs out of the destination",
		"dockerctl/../../escaped.go": "climbs out after cleaning",
		"..":                         "is the parent itself",
		"":                           "names nothing",
		"/absolute/escaped.go":       "is rooted (not IsAbs on Windows)",
		`dockerctl\..\..\escaped.go`: "climbs out through Windows separators",
		"C:/escaped.go":              "carries a Windows volume",
	} {
		t.Run("refuses "+name+" — "+why, func(t *testing.T) {
			_, err := safeJoin(dest, name)
			require.ErrorIs(t, err, errUnsafePath)
		})
	}
}

// assertAbsent fails when path exists. testify/v2 has FileExists and no negative form of it.
func assertAbsent(tb testing.TB, path string) {
	tb.Helper()

	_, err := os.Stat(path)
	assert.Truef(tb, errors.Is(err, fs.ErrNotExist), "expected %q not to exist, stat says: %v", path, err)
}

// writeArchive builds a gzipped tar of the given entries, so the extraction is tested against an
// archive rather than against a mock of one.
func writeArchive(tb testing.TB, path string, entries map[string]string) {
	tb.Helper()

	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for name, content := range entries {
		require.NoError(tb, tw.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     int64(fileMode),
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}))
		_, err := tw.Write([]byte(content))
		require.NoError(tb, err)
	}

	require.NoError(tb, tw.Close())
	require.NoError(tb, gz.Close())
	require.NoError(tb, os.WriteFile(path, buf.Bytes(), fileMode))
}
