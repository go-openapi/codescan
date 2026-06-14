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

// TestCoverage_Bug3007 locks the fix for go-swagger issue #3007
// ("[Bug]generate spec error"): a non-swagger kubebuilder marker
// (`+kubebuilder:default:=false`) used to abort the scan with
// `strconv.ParseBool parsing "=false"`. The scan now succeeds.
//
// The residual (the marker leaking into the field description) is now closed
// alongside go-swagger#2687: the lexer drops `+marker`-style directive lines
// from prose, so the description is clean.
func TestCoverage_Bug3007(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3007/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err, "a kubebuilder marker must not abort the scan")
	require.NotNil(t, doc)

	enabled := doc.Definitions["Thing"].Properties["enabled"]
	assert.Equal(t, "boolean", enabled.Type[0])
	// FIXED (with #2687): the non-swagger marker is stripped from the prose.
	assert.Equal(t, "Enabled flag.", enabled.Description)
	assert.NotContains(t, enabled.Description, "+kubebuilder")

	scantest.CompareOrDumpJSON(t, doc, "bugs_3007_schema.json")
}
