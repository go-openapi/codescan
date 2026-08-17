// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"go/build"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan/internal/packages/vfs"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The standard library vendors a handful of golang.org/x packages, and crypto/tls imports one of
// them by its real path rather than by a std one.
//
// Nothing else in the resolver can find it: the stdlib branch is gated on an import path whose first
// segment has no dot, and every other branch answers for the module under scan.
const (
	stdVendored = "golang.org/x/crypto/chacha20poly1305"
	stdImporter = "crypto/tls"
)

// newGorootResolver builds a resolver over the real Go installation, which is the only place a
// standard library with a vendor directory can be found.
func newGorootResolver(t *testing.T) *Resolver {
	t.Helper()

	ctx := build.Default
	r, err := NewResolver(Config{Dir: t.TempDir(), Context: &ctx, FS: vfs.New(nil)})
	require.NoError(t, err)
	require.NotEmpty(t, r.srcRoot)

	if !r.vfs.IsDir(filepath.Join(r.srcRoot, "vendor", stdVendored)) {
		t.Skipf("this Go installation does not vendor %s into its standard library", stdVendored)
	}

	return r
}

// TestResolveImportFrom_StdVendor covers the second vendor root: the standard library's own.
//
// go mod vendor flattens the whole module graph into the MAIN module's vendor directory, and a
// dependency's own is never read - so one vendor root answers for module mode entirely. std is the
// exception, because std never goes through modload: the go command resolves its imports with the
// older walk that climbs from the importing package to the source root, which is how a package in
// GOROOT/src reaches GOROOT/src/vendor and nothing outside it does.
func TestResolveImportFrom_StdVendor(t *testing.T) {
	t.Parallel()

	r := newGorootResolver(t)

	t.Run("an importer inside GOROOT reaches the vendored copy", func(t *testing.T) {
		dir, pkgPath, ok := r.ResolveImportFrom(stdVendored, filepath.Join(r.srcRoot, stdImporter))
		require.True(t, ok, "crypto/tls imports this, so it has to resolve")
		assert.Equal(t, filepath.Join(r.srcRoot, "vendor", stdVendored), dir)

		// The path go list reports, so that the two loaders agree on identity: a type recognized under
		// one name and not the other is the whole failure mode this prefix exists to avoid.
		assert.Equal(t, "vendor/"+stdVendored, pkgPath)
	})

	// The bound is the point. Without it the Go installation's pinned copy would answer for anybody
	// whose own module graph came up short, which is a build the go command refuses outright.
	t.Run("an importer outside GOROOT does not", func(t *testing.T) {
		_, _, ok := r.ResolveImportFrom(stdVendored, t.TempDir())
		assert.False(t, ok)
	})

	t.Run("and neither does one that names no importer", func(t *testing.T) {
		_, _, ok := r.ResolveImport(stdVendored)
		assert.False(t, ok)
	})

	// A std import from a std importer still resolves the ordinary way: the vendor lookup is tried
	// first and has nothing to say about it.
	t.Run("a std import is unaffected", func(t *testing.T) {
		dir, pkgPath, ok := r.ResolveImportFrom("encoding/json", filepath.Join(r.srcRoot, stdImporter))
		require.True(t, ok)
		assert.Equal(t, filepath.Join(r.srcRoot, "encoding/json"), dir)
		assert.Equal(t, "encoding/json", pkgPath)
	})
}
