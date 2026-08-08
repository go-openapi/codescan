// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build cgo

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// A program that needs cgo to BUILD must still be scannable.
//
// codescan reads declarations; it never compiles, and it does not run the cgo tool. That is fine for
// the C code itself — nobody wants a C struct in their API — but it must not be a reason to refuse the
// Go declarations sitting next to it. Requiring authors to give up cgo to get a spec is not a trade
// this scanner is entitled to ask for.
func TestCgo_ProgramRequiringCgoStillScans(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0o600))
	}
	write("go.mod", "module example.com/cgoapi\n\ngo 1.25.0\n")
	write("api.go", `package cgoapi

/*
#include <stdlib.h>
typedef struct { int id; } widget_t;
*/
import "C"

// Widget mixes ordinary Go fields with cgo-typed ones.
//
// swagger:model widget
type Widget struct {
	// The name
	Name string `+"`json:\"name\"`"+`
	// A C-typed field
	Size C.int `+"`json:\"size\"`"+`
	// A pointer to a C struct
	Raw *C.widget_t `+"`json:\"raw\"`"+`
}

func alloc() { _ = C.malloc(1) }
`)

	var hints, others int
	doc, err := codescan.Run(&codescan.Options{
		Packages:            []string{"."},
		WorkDir:             root,
		ScanModels:          true,
		ToolchainFreeLoader: true,
		OnDiagnostic: func(d codescan.Diagnostic) {
			if d.Code == "scan.synthesized-import" {
				hints++

				return
			}
			others++
		},
	})
	require.NoError(t, err, "a cgo program must produce a spec, not an error")

	def, ok := doc.Definitions["widget"]
	require.True(t, ok)

	// The Go fields are unaffected: this is the part an author actually documents.
	assert.Contains(t, def.Properties, "name")
	assert.Equal(t, []string{"string"}, []string(def.Properties["name"].Type))

	// The C-typed ones survive as untyped members rather than sinking the scan. Their documentation is
	// still carried, which is more than the go command manages — it emits a $ref to a generated
	// _Ctype_int definition and drops the description with it.
	require.Contains(t, def.Properties, "size")
	assert.Equal(t, "A C-typed field", def.Properties["size"].Description)
	assert.Empty(t, def.Properties["size"].Type, "nothing is known about a C type without running cgo")

	assert.Equal(t, 1, hints, "cgo is announced once, as a hint")
	assert.Zero(t, others,
		"and nothing else: a package that merely uses cgo has not 'failed to type-check'")
}

// The cgo exemption is scoped to cgo files. A genuine mistake elsewhere in the same package is still
// the author's to fix, and still reported with its position.
func TestCgo_RealErrorsElsewhereSurvive(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0o600))
	}
	write("go.mod", "module example.com/mixed\n\ngo 1.25.0\n")
	write("cgofile.go", "package mixed\n\n/*\n#include <stdlib.h>\n*/\nimport \"C\"\n\ntype WithC struct{ Size C.int }\n")
	write("plain.go", "package mixed\n\nvar Broken int = \"not an int\"\n")

	var degraded []string
	_, err := codescan.Run(&codescan.Options{
		Packages:            []string{"."},
		WorkDir:             root,
		ToolchainFreeLoader: true,
		OnDiagnostic: func(d codescan.Diagnostic) {
			if d.Code == "scan.degraded-load" {
				degraded = append(degraded, d.Message)
			}
		},
	})
	require.NoError(t, err)

	require.Len(t, degraded, 1, "the ordinary type error is still reported")
	assert.Contains(t, degraded[0], "plain.go", "and it names the file it is in")
	assert.NotContains(t, degraded[0], "package C", "while the cgo noise stays suppressed")
}
