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

// TestCoverage_Bug2638 locks the fix for go-swagger issue #2638: a field group with multiple names
// on one line (`R, G, B, A uint8`) emits one property per name, not a single mislabeled property.
func TestCoverage_Bug2638(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2638/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	props := doc.Definitions["RGBA"].Properties
	require.Len(t, props, 4, "one property per name in the field group")
	for _, name := range []string{"R", "G", "B", "A"} {
		ps, ok := props[name]
		require.Truef(t, ok, "property %q must be present", name)
		assert.Equal(t, []string{"integer"}, []string(ps.Type))
		// name == Go name, so no x-go-name override leaks in.
		assert.NotContains(t, ps.Extensions, "x-go-name", "property %q must not carry a stray x-go-name", name)
	}

	// Mixed: the multi-name group keeps each member; the json rename is dropped (a single rename
	// cannot name X/Y/Z) while omitempty still applied during parsing.
	// Single-name fields are unaffected.
	mixed := doc.Definitions["Mixed"].Properties
	require.Len(t, mixed, 4, "X, Y, Z each emit, plus Label")
	for _, name := range []string{"X", "Y", "Z", "Label"} {
		_, ok := mixed[name]
		assert.Truef(t, ok, "property %q must be present", name)
	}
	_, renamed := mixed["renamed"]
	assert.False(t, renamed, "a single json rename must not collapse the multi-name group")
}
