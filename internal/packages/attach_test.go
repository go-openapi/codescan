// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package packages

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The go/packages strategy cannot decide per dependency during the load — a LoadMode is one value for
// the whole of it — so the decision is made afterwards, on the graph it hands back.
//
// This pins both halves of that decision against a real dependency of each kind: strfmt, which marks
// its own types and therefore has something to say, and the standard library, which does not.
func TestAttachAnnotatedDependencies(t *testing.T) {
	t.Parallel()

	reasons := map[string]string{}
	l := NewLoader(
		WithCompiledDependencies(),
		WithOnExportOnly(func(e ExportOnly) { reasons[e.Path] = e.Reason }),
	)

	roots, err := l.Load(&Config{Dir: "../../fixtures/goparsing"}, "./petstore/...")
	require.NoError(t, err)
	require.NotEmpty(t, roots)

	graph := map[string]*Package{}
	var walk func(*Package)
	walk = func(pkg *Package) {
		if pkg == nil || graph[pkg.PkgPath] != nil {
			return
		}
		graph[pkg.PkgPath] = pkg
		for _, imp := range pkg.Imports {
			walk(imp)
		}
	}
	for _, root := range roots {
		walk(root)
	}

	t.Run("a dependency that carries annotations is read back", func(t *testing.T) {
		strfmt := graph["github.com/go-openapi/strfmt"]
		require.NotNil(t, strfmt, "the petstore reaches strfmt through its models")

		require.NotEmpty(t, strfmt.Syntax, "its files carry `swagger:strfmt`, so they are parsed")
		require.NotNil(t, strfmt.TypesInfo)
		require.NotEmpty(t, strfmt.TypesInfo.Defs)

		// The join is by name against the export-data scope, so a parsed declaration and the object the
		// compiler wrote down have to be the same object — not merely two types that look alike.
		var found bool
		for ident, obj := range strfmt.TypesInfo.Defs {
			if ident.Name != "DateTime" {
				continue
			}
			found = true
			assert.Same(t, strfmt.Types.Scope().Lookup("DateTime"), obj,
				"the bridged object is the one export data already held")
		}
		assert.True(t, found, "DateTime is declared in strfmt's source and exported, so it bridges")

		assert.NotContains(t, reasons, "github.com/go-openapi/strfmt",
			"nothing was given up here, so there is nothing to announce")
	})

	t.Run("a dependency that carries none is left alone, and said so", func(t *testing.T) {
		fmtPkg := graph["fmt"]
		require.NotNil(t, fmtPkg)

		assert.Empty(t, fmtPkg.Syntax, "no marker, so its source is never parsed")
		assert.Nil(t, fmtPkg.TypesInfo)
		assert.NotNil(t, fmtPkg.Types, "its types are complete all the same — that is the whole point")

		assert.Equal(t, "nothing in its source is annotated, so it was not parsed", reasons["fmt"],
			"recorded rather than announced: it only matters if some builder later wants a declaration from here")
	})

	t.Run("the roots are untouched", func(t *testing.T) {
		for _, root := range roots {
			assert.NotEmpty(t, root.Syntax)
			require.NotNil(t, root.TypesInfo)
			assert.NotEmpty(t, root.TypesInfo.Types,
				"a root is loaded whole, so it keeps the expression map an attached dependency never has")
		}
	})
}
