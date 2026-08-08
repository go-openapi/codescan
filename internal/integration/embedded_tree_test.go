// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package integration_test

import (
	"bytes"
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

// The configuration an embedded tree can actually be built out of.
//
// Options.FS was meant first for embed.FS, and what an embed.FS can hold decides the rest: a module's
// own source and its vendor directory, because both sit inside the module. What it cannot hold is
// GOROOT — nothing embeds the standard library's source — and neither GOROOT nor the module cache is
// inside a module to begin with.
//
// So the two halves are supplied differently, and this pins that they compose:
//
//	source + vendored dependencies   from FS, read and parsed like any other tree
//	the standard library             from ExportData, which is itself an fs.FS and embeds too
//
// The vendored dependency here carries its own `swagger:strfmt`, which is the property worth pinning:
// it is read from inside the embedded tree, so an embedded scan keeps the marks that a compiled-away
// dependency would have lost.
const (
	embeddedAppMod = "module example.com/app\n\ngo 1.25.0\n\nrequire example.com/dep v1.0.0\n"

	embeddedAppSrc = `package app

import (
	"time"

	"example.com/dep"
)

// Event is the model under scan.
//
// swagger:model Event
type Event struct {
	// When the event happened.
	When time.Time ` + "`json:\"when\"`" + `

	// On which day.
	On dep.Day ` + "`json:\"on\"`" + `
}
`

	// The vendored dependency marks its own type, exactly as go-openapi's strfmt does.
	embeddedDepSrc = `package dep

// Day is a calendar date.
//
// swagger:strfmt date
type Day struct{ y, m, d int }
`

	embeddedVendorList = "# example.com/dep v1.0.0\n## explicit; go 1.25\nexample.com/dep\n"

	// Stands in for the real time package: only the shape the scan reads matters.
	embeddedStdTime = "package time\n\ntype Time struct{ wall uint64 }\n"
)

// embeddedExportData builds the standard-library half as a blob, the way a real one is produced by
// hack/genexportdata and then embedded next to the source.
func embeddedExportData(tb testing.TB) fstest.MapFS {
	tb.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "time.go", embeddedStdTime, parser.ParseComments)
	require.NoError(tb, err)

	pkg, err := (&types.Config{}).Check("time", fset, []*ast.File{f}, nil)
	require.NoError(tb, err)

	var blob bytes.Buffer
	require.NoError(tb, gcexportdata.Write(&blob, fset, pkg))

	return fstest.MapFS{"time.export": &fstest.MapFile{Data: blob.Bytes()}}
}

// embeddedTree is what an embed.FS of a vendored module looks like: everything under the module root,
// nothing above it.
func embeddedTree() fstest.MapFS {
	return fstest.MapFS{
		"go.mod":                        &fstest.MapFile{Data: []byte(embeddedAppMod)},
		"app/app.go":                    &fstest.MapFile{Data: []byte(embeddedAppSrc)},
		"vendor/modules.txt":            &fstest.MapFile{Data: []byte(embeddedVendorList)},
		"vendor/example.com/dep/dep.go": &fstest.MapFile{Data: []byte(embeddedDepSrc)},
	}
}

func TestEmbeddedTree_SourceAndVendorFromFS_StdlibFromExportData(t *testing.T) {
	t.Parallel()

	var degraded, synthesized int
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./app"},
		WorkDir:    ".",
		ScanModels: true,
		FS:         embeddedTree(),
		ExportData: embeddedExportData(t),
		GOOS:       "linux",
		GOARCH:     "amd64",
		OnDiagnostic: func(d codescan.Diagnostic) {
			switch d.Code {
			case "scan.degraded-load":
				degraded++
			case "scan.synthesized-import":
				synthesized++
			default:
				// Everything else is beside the point here: this test is about what the tree could and
				// could not supply, and those are the two codes that say so.
			}
		},
	})
	require.NoError(t, err)

	assert.Zero(t, synthesized, "nothing had to be invented: the tree holds the module and its vendor, the blob holds std")
	assert.Zero(t, degraded, "and so the load is not degraded")

	event, ok := doc.Definitions["Event"]
	require.True(t, ok)

	when, ok := event.Properties["when"]
	require.True(t, ok)
	assert.Equal(t, "date-time", when.Format,
		"time.Time is recognized from its identity, which export data carries as faithfully as source does")

	on, ok := event.Properties["on"]
	require.True(t, ok)
	assert.Equal(t, "date", on.Format,
		"the vendored dependency's own swagger:strfmt is inside the embedded tree, so it is read")
}
