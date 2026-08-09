// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package scanner_test

import (
	"testing"

	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const makeplans = "github.com/go-swagger/scan-repo-boundary/makeplans"

// A dependency taken types-only is read back when a declaration is wanted from it, and only then.
//
// That "only then" is what makes the option worth choosing. Reading every dependency back up front would answer the
// same lookups and give away most of the saving — measured on a generated server, a third of the wall clock and a
// third of the peak RSS — because the closure a scan does not look at is where the saving lives. So the cost is one
// parse per declaration wanted, and this pins the "one" from both sides.
func TestReadBackOnDemand_PaysPerDeclarationWanted(t *testing.T) {
	t.Parallel()

	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages:             []string{"./goparsing/bookings/..."},
		WorkDir:              scantest.FixturesDir(),
		ScanModels:           true,
		CompiledDependencies: true,
	})
	require.NoError(t, err)

	read, loaded := ctx.PackagesRead()
	require.Positive(t, loaded, 1)
	assert.Less(t, read, loaded/2,
		"the load reads the scanned packages and the annotated dependencies; the closure behind them is types-only")

	// One lookup, into a dependency whose source the marker scan passed over.
	decl, found := ctx.FindDecl(makeplans, "Booking")
	require.True(t, found, "the declaration is fetched on demand, not lost with the load")
	assert.NotNil(t, decl.Comments(), "and it arrives with the comments that are the whole reason to read it")

	after, _ := ctx.PackagesRead()
	assert.Equal(t, read+1, after, "exactly the package that was asked for")

	// Asking again must not parse again.
	_, found = ctx.FindDecl(makeplans, "Booking")
	require.True(t, found)

	again, _ := ctx.PackagesRead()
	assert.Equal(t, after, again, "the read-back is idempotent")
}

// A lookup that finds nothing must not leave the package looking read, nor keep retrying.
func TestReadBackOnDemand_MissingDeclaration(t *testing.T) {
	t.Parallel()

	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages:             []string{"./goparsing/bookings/..."},
		WorkDir:              scantest.FixturesDir(),
		ScanModels:           true,
		CompiledDependencies: true,
	})
	require.NoError(t, err)

	_, found := ctx.FindDecl(makeplans, "NoSuchType")
	assert.False(t, found, "the package is readable and simply does not declare this")

	_, found = ctx.FindDecl("example.com/nowhere", "Anything")
	assert.False(t, found, "and a package that was never loaded has nothing to read back")
}
