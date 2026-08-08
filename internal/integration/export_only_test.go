// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package integration

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
	"testing/fstest"

	"golang.org/x/tools/go/gcexportdata"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// A dependency whose types arrive without its source is a fact about the LOAD. Whether it matters is
// a fact about the SPEC, and the two are not the same question.
//
// Announcing it at load time announces it for every package the roots reach — which includes
// everything imported from inside a function body. Scanning a generated client under a WebAssembly
// guest, where the whole standard library arrives this way, that is hundreds of notices about
// packages like `strconv` that no part of the API surface will ever touch, and the handful that
// matter are lost among them.
//
// So the notice is raised from the lookup that wanted a declaration and did not get one. These cases
// pin the four outcomes that distinction has to produce.
const (
	sourcelessLib  = "package lib\n\ntype Stamp struct{ Seconds int64 }\n"
	sourcelessTime = "package time\n\ntype Time struct{ wall uint64 }\ntype Duration int64\n"
)

// exportOnlyFor writes export data for the two packages these cases serve without source.
//
// Neither is on the virtual filesystem, so the loader can only know their types — the state a scan is
// in under a WebAssembly guest, where the standard library ships as export data and there is no
// GOROOT to read.
func exportOnlyFor(t *testing.T) fstest.MapFS {
	t.Helper()

	write := func(pkgPath, src string) []byte {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
		require.NoError(t, err)

		pkg, err := (&types.Config{}).Check(pkgPath, fset, []*ast.File{f}, nil)
		require.NoError(t, err)

		var blob bytes.Buffer
		require.NoError(t, gcexportdata.Write(&blob, fset, pkg))

		return blob.Bytes()
	}

	return fstest.MapFS{
		"example.com/lib.export": &fstest.MapFile{Data: write("example.com/lib", sourcelessLib)},
		"time.export":            &fstest.MapFile{Data: write("time", sourcelessTime)},
	}
}

func TestExportOnly_ReportedWhereItCosts(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		app  string
		want string // the type the notice must name, or "" for silence
	}{
		{
			// The `strconv` case. The package is in the closure because a function body reaches it, and
			// nothing inside a body can reach the emitted document. Announcing this is the noise.
			name: "reached only from a function body",
			app: "package app\n\nimport \"example.com/lib\"\n\n// swagger:model Model\n" +
				"type Model struct {\n\tName string `json:\"name\"`\n}\n\nfunc use() { _ = lib.Stamp{} }\n",
			want: "",
		},
		{
			// A field the document carries. Whatever the declaration said about itself — a
			// swagger:strfmt, its godoc — is absent, and nothing else in the output shows the gap.
			name: "a field the API surface carries",
			app: "package app\n\nimport \"example.com/lib\"\n\n// swagger:model Model\n" +
				"type Model struct {\n\tAt lib.Stamp `json:\"at\"`\n}\n",
			want: "example.com/lib.Stamp",
		},
		{
			// Recognised from its identity alone, ahead of any declaration lookup — so nothing was
			// wanted from the source and nothing was lost. Silence here is the point of the change.
			name: "a recognised type needs no source",
			app: "package app\n\nimport \"time\"\n\n// swagger:model Model\n" +
				"type Model struct {\n\tAt time.Time `json:\"at\"`\n}\n",
			want: "",
		},
		{
			// The complement, and the case worth a notice: consumed, not recognised. What a
			// time.Duration should look like on the wire is the author's decision — which is why
			// go-openapi offers strfmt.Duration — so codescan says it cannot tell rather than guessing.
			name: "a consumed type with no recogniser",
			app: "package app\n\nimport \"time\"\n\n// swagger:model Model\n" +
				"type Model struct {\n\tD time.Duration `json:\"d\"`\n}\n",
			want: "time.Duration",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tree := fstest.MapFS{
				"go.mod": &fstest.MapFile{Data: []byte(
					"module example.com/app\n\ngo 1.25.0\n\nrequire example.com/lib v1.0.0\n")},
				"app/app.go": &fstest.MapFile{Data: []byte(tc.app)},
			}

			var sourceless []string
			_, err := codescan.Run(&codescan.Options{
				Packages:            []string{"./app"},
				WorkDir:             ".",
				ScanModels:          true,
				ToolchainFreeLoader: true,
				FS:                  tree,
				ExportData:          exportOnlyFor(t),
				GOOS:                "linux",
				GOARCH:              "amd64",
				OnDiagnostic: func(d codescan.Diagnostic) {
					if strings.Contains(d.Message, "could not be read") {
						sourceless = append(sourceless, d.Message)
					}
				},
			})
			require.NoError(t, err)

			if tc.want == "" {
				assert.Empty(t, sourceless,
					"nothing in the document reached this package's source, so there is nothing to report")

				return
			}

			require.Len(t, sourceless, 1, "the notice names the declaration that was wanted, once")
			assert.Contains(t, sourceless[0], tc.want)
		})
	}
}
