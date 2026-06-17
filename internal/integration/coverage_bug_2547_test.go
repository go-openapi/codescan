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

// TestCoverage_Bug2547 asserts the EXPECTED resolution of go-swagger issue #2547
// (quirk F8): a string example/default written with surrounding quotes must have
// those delimiter quotes stripped. `example: ""` is the empty string; the
// scanner currently keeps the quotes verbatim (`""` -> the 2-char string `""`,
// `"Foo"` -> `"Foo"` with quotes), so this is currently RED.
func TestCoverage_Bug2547(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2547/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	props := doc.Definitions["Thing"].Properties
	assert.Equal(t, "", props["label"].Example,
		"a quoted empty-string example must be the empty string, not `\"\"` (go-swagger#2547/F8)")
	assert.Equal(t, "Foo", props["title"].Example,
		"surrounding quotes on a string example must be stripped")
	assert.Equal(t, "bar", props["title"].Default,
		"surrounding quotes on a string default must be stripped")
}
