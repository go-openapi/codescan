// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
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
// Residual (flagged 📖 Need doc): the marker is currently absorbed into the
// field description rather than ignored. Filtering known non-swagger markers
// (kubebuilder, etc.) out of prose is a possible future enhancement; this
// test documents the current behaviour so a change to it is conscious.
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
	// Current behaviour: the non-swagger marker leaks into the description.
	assert.True(t, strings.Contains(enabled.Description, "+kubebuilder:default:=false"),
		"TODO #3007: kubebuilder marker currently absorbed into the description (future: filter non-swagger markers)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_3007_schema.json")
}
