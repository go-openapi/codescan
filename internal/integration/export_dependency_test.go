// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
	"testing/fstest"

	"golang.org/x/tools/go/gcexportdata"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// A dependency carrying every shape that reads types.Info.Types downstream.
//
// Widget reaches the unguarded lookup in builders/schema (a named type over a struct, no override
// classifier claims it). Vehicle and Car reach the guarded one in builders/spec, where the reverse
// allOf index decides whether a discriminated subtype is discovered at all — and the subtype has to
// live in the dependency, since that lookup is made against the SUBTYPE's package.
const dependencySource = `package lib

// Widget is a plain model declared in a dependency.
//
// swagger:model libWidget
type Widget struct {
	Name string ` + "`json:\"name\"`" + `
}

// Vehicle is a discriminated base declared in a dependency.
//
// swagger:model libVehicle
type Vehicle interface {
	// The kind of vehicle
	//
	// discriminator: true
	// swagger:name kind
	Kind() string
}

// Car is a subtype of Vehicle, declared in the same dependency.
//
// It is reached only through the reverse allOf index.
//
// swagger:model libCar
type Car struct {
	// swagger:allOf
	Vehicle

	// How many wheels
	Wheels int64 ` + "`json:\"wheels\"`" + `
}
`

// exportDataFor type-checks src and writes it out the way the compiler would, so the test needs no
// toolchain, no GOROOT and no build cache.
func exportDataFor(t *testing.T, pkgPath, src string) fstest.MapFS {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "lib.go", src, parser.ParseComments)
	require.NoError(t, err)

	pkg, err := (&types.Config{}).Check(pkgPath, fset, []*ast.File{f}, nil)
	require.NoError(t, err)

	var blob bytes.Buffer
	require.NoError(t, gcexportdata.Write(&blob, fset, pkg))

	return fstest.MapFS{pkgPath + ".export": &fstest.MapFile{Data: blob.Bytes()}}
}

// dependencyTree is an app whose models come from a module beside it.
func dependencyTree(appSource string) fstest.MapFS {
	return fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte(
			"module example.com/app\n\ngo 1.25.0\n\nrequire example.com/lib v1.0.0\n\nreplace example.com/lib => ./lib\n")},
		"lib/go.mod": &fstest.MapFile{Data: []byte("module example.com/lib\n\ngo 1.25.0\n")},
		"lib/lib.go": &fstest.MapFile{Data: []byte(dependencySource)},
		"app/app.go": &fstest.MapFile{Data: []byte(appSource)},
	}
}

// scanBothWays runs the same tree with and without export data and returns the two documents.
func scanBothWays(t *testing.T, appSource string, scanModels bool) (fromSource, fromExport []byte) {
	t.Helper()

	tree := dependencyTree(appSource)
	exports := exportDataFor(t, "example.com/lib", dependencySource)

	run := func(withExport bool) []byte {
		opts := &codescan.Options{
			Packages:            []string{"./app"},
			WorkDir:             ".",
			ScanModels:          scanModels,
			ToolchainFreeLoader: true,
			FS:                  tree,
			GOOS:                "linux",
			GOARCH:              "amd64",
		}
		if withExport {
			opts.ExportData = exports
		}

		doc, err := codescan.Run(opts)
		require.NoError(t, err, "a scan with export data must not fail where one without it succeeds")

		blob, err := json.Marshal(doc)
		require.NoError(t, err)

		return blob
	}

	return run(false), run(true)
}

// Export data is an optimisation, so it has to be invisible in the output.
//
// It was not. A dependency served from export data was attached with its syntax parsed but never
// type-checked, which gave the builders declarations to read and no record of what those declarations
// denote: builders/schema asserts that record exists and panicked on the zero value, taking the whole
// scan with it. There is no way to fill that record in afterwards — go/types marks a type expression
// with an unexported field — so the choice is now made per package and made whole.
func TestExportDependency_ScanIsUnchanged(t *testing.T) {
	t.Parallel()

	const app = `package app

import "example.com/lib"

// Holder uses the dependency's types.
//
// swagger:model holder
type Holder struct {
	W lib.Widget  ` + "`json:\"w\"`" + `
	V lib.Vehicle ` + "`json:\"v\"`" + `
}
`

	fromSource, fromExport := scanBothWays(t, app, true)
	assert.JSONEq(t, string(fromSource), string(fromExport),
		"the same tree must produce the same spec whether or not dependency types came precompiled")
}

// The second reader of that record decides whether a discriminated subtype is discovered at all, and
// it fails quietly rather than loudly: a missing entry means the subtype is not indexed against its
// base, and it simply never appears.
//
// ScanModels is off here on purpose. With it on every model is emitted anyway and the index proves
// nothing; off, the subtype can only arrive by being pulled in behind the base it extends.
func TestExportDependency_DiscriminatedSubtypeIsStillDiscovered(t *testing.T) {
	t.Parallel()

	const app = `package app

import "example.com/lib"

// swagger:route GET /vehicles vehicles listVehicles
//
// Lists vehicles.
//
//	Responses:
//	  200: vehicleResponse

// vehicleResponse is the response.
//
// swagger:response vehicleResponse
type vehicleResponse struct {
	// in: body
	Body lib.Vehicle
}
`

	fromSource, fromExport := scanBothWays(t, app, false)

	var source, export map[string]any
	require.NoError(t, json.Unmarshal(fromSource, &source))
	require.NoError(t, json.Unmarshal(fromExport, &export))

	defs, ok := source["definitions"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, defs, "libCar",
		"precondition: the subtype is discovered behind its base when the dependency is read from source")

	assert.JSONEq(t, string(fromSource), string(fromExport),
		"and it is still discovered when that dependency's types came precompiled")
}
