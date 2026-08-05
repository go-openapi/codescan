// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build cgo

package integration_test

import (
	"testing"
	"testing/fstest"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestScanVirtualFSScansCgoFiles is the toolchain-free loader's half of go-swagger#1096: annotations
// in a file that imports "C" must still be read.
//
// go/build keeps such files in CgoFiles rather than GoFiles, because compiling them needs the cgo
// tool first. codescan never compiles, so a loader that reads only GoFiles drops the declaration and
// emits an empty spec — the exact shape of #1096, reintroduced.
//
// Tagged cgo like the go/packages counterpart in coverage_bug_1096_test.go: under CGO_ENABLED=0 the
// build constraint is unsatisfied and there is no behaviour to assert.
func TestScanVirtualFSScansCgoFiles(t *testing.T) {
	t.Parallel()

	tree := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/api\n\ngo 1.25.0\n")},
		"orders/order.go": &fstest.MapFile{Data: []byte(`package orders

/*
#include <stdlib.h>
*/
import "C"

// Order is used to foobar.
//
// swagger:model order
type Order struct {
	// Name of the order
	Name string ` + "`json:\"name\"`" + `
}

func alloc() { _ = C.malloc(1) }
`)},
	}

	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./orders"},
		WorkDir:    ".",
		ScanModels: true,
		FS:         tree,
		GOOS:       "linux",
		GOARCH:     "amd64",
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	def, ok := doc.Definitions["order"]
	require.True(t, ok, "the annotated type lives in a cgo file and must still be discovered, got %v",
		definitionNames(doc.Definitions))
	assert.Contains(t, def.Properties, "name")
}
