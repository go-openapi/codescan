// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliconf

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// write puts a configuration file at dir/name and reports its path.
func write(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("scan:\n  workdir: .\n"), 0o600))

	return path
}

// discover parses argv into the configuration flags and asks them where the file is.
func discover(t *testing.T, start string, argv ...string) (string, error) {
	t.Helper()

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	flags := Register(fs)
	require.NoError(t, fs.Parse(argv))

	return flags.Discover(start)
}

func TestDiscoverFindsAFileWhereItStands(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	want := write(t, dir, Names[0])

	got, err := discover(t, dir)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestDiscoverWalksUp is what lets a command be run from anywhere inside a project.
func TestDiscoverWalksUp(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	want := write(t, root, Names[0])

	deep := filepath.Join(root, "internal", "api", "handlers")
	require.NoError(t, os.MkdirAll(deep, 0o750))

	got, err := discover(t, deep)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestDiscoverStopsAtTheNearest states that finding one is the end of it: a file further up does
// not half-override the one beside you.
func TestDiscoverStopsAtTheNearest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write(t, root, Names[0])

	nearer := filepath.Join(root, "api")
	require.NoError(t, os.MkdirAll(nearer, 0o750))
	want := write(t, nearer, Names[0])

	got, err := discover(t, nearer)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiscoverPrefersTheNamesInOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	want := write(t, dir, Names[0])
	write(t, dir, Names[1])

	got, err := discover(t, dir)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestDiscoverFindingNothingIsNotAFailure: most projects have no configuration file, and that is
// not a thing to complain about.
func TestDiscoverFindingNothingIsNotAFailure(t *testing.T) {
	t.Parallel()

	got, err := discover(t, t.TempDir())

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestDiscoverIgnoresADirectoryNamedLikeAConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, Names[0]), 0o750))

	got, err := discover(t, dir)

	require.NoError(t, err)
	assert.Empty(t, got, "a directory is not a file, whatever it is called")
}

func TestDiscoverTakesTheFileItIsGiven(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	want := write(t, dir, "elsewhere.yaml")

	got, err := discover(t, t.TempDir(), "-config", want)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestDiscoverRefusesAFileThatIsNotThere: a caller who named a file meant that file, and quietly
// searching for another one answers a question they did not ask.
func TestDiscoverRefusesAFileThatIsNotThere(t *testing.T) {
	t.Parallel()

	_, err := discover(t, t.TempDir(), "-config", filepath.Join(t.TempDir(), "absent.yaml"))

	require.ErrorIs(t, err, ErrBadConfig)
	assert.Contains(t, err.Error(), "absent.yaml")
}

// TestDiscoverNoConfigFindsNothingEvenWhereThereIsSomething is the reproducible-run escape hatch.
func TestDiscoverNoConfigFindsNothingEvenWhereThereIsSomething(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write(t, dir, Names[0])

	got, err := discover(t, dir, "-no-config")

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestDiscoverTakesTheShortFlag(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	want := write(t, dir, "elsewhere.yaml")

	got, err := discover(t, t.TempDir(), "-c", want)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestDiscoverAcceptsBothSpellingsOfTheSameFile: -c is -config, so saying both is repetition rather
// than a contradiction.
func TestDiscoverAcceptsBothSpellingsOfTheSameFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	want := write(t, dir, "elsewhere.yaml")

	got, err := discover(t, t.TempDir(), "-config", want, "-c", want)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestDiscoverRefusesTwoDifferentFiles is why the two spellings are stored apart: sharing one
// variable would have let whichever the parser read last win in silence.
func TestDiscoverRefusesTwoDifferentFiles(t *testing.T) {
	t.Parallel()

	_, err := discover(t, t.TempDir(), "-config", "one.yaml", "-c", "another.yaml")

	require.ErrorIs(t, err, ErrBadConfig)
	assert.Contains(t, err.Error(), "one.yaml")
	assert.Contains(t, err.Error(), "another.yaml")
}

// TestDiscoverRefusesAFileAndNoFileAtOnce covers the contradiction the pair of flags makes possible.
//
// Refused rather than resolved: there is no reading of "use this file, and use no file" that the
// caller can have meant, and picking one would be inventing an intention.
func TestDiscoverRefusesAFileAndNoFileAtOnce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := write(t, dir, Names[0])

	_, err := discover(t, dir, "-c", path, "-no-config")

	require.ErrorIs(t, err, ErrBadConfig)
	assert.Contains(t, err.Error(), NoFlag)
}
