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

// TestCoverage_Bug2409 locks go-swagger issue #2409 ("annotate structures with extensions"): a
// TYPE-level Extensions block emits its x-* extensions at the definition level. (The reporter's
// specific x-go-type IMPORT semantics is a separate design question — poison-queue #2924 — but
// the extension mechanism itself works for any x-* key.)
func TestCoverage_Bug2409(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2409/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	mt := doc.Definitions["MyType"]
	assert.Equal(t, "hello", mt.Extensions["x-custom-type-ext"],
		"a type-level Extensions block emits at the definition level (go-swagger#2409)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2409_schema.json")
}
