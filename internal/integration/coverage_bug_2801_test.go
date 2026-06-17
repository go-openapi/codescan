// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug2801 locks the fix for go-swagger issue #2801 ("Spec is not
// generated if generic struct declaration is not in the same file as
// swagger:parameters"): the generic struct (WrappedRequest[T], in types.go) is
// resolved even though its swagger:parameters instantiation lives in another
// file of the same package (params.go). The v0.30 empty-spec is gone.
func TestCoverage_Bug2801(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2801/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	_, ok := doc.Definitions["Request"]
	assert.True(t, ok, "the cross-file generic's type argument must be resolved and emitted")
	require.NotNil(t, doc.Paths)
	assert.Contains(t, doc.Paths.Paths, "/op")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2801_schema.json")
}
