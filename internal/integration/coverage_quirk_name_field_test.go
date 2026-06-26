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

// TestQuirk_NameOnField verifies the fix for quirk F5 (doc-site-quirks.md): swagger:name on a
// struct field now overrides the emitted property name, just as it already did on interface
// methods.
//
// Previously the override was scanned but silently dropped — the property kept its Go field name
// or json-tag name.
//
// The Go field name is still recorded as x-go-name when it differs from the emitted JSON name, so
// the override does not lose traceability.
func TestQuirk_NameOnField(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./quirks/name-on-field/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	props := doc.Definitions["Account"].Properties
	require.NotNil(t, props)

	// swagger:name with no json tag → property renamed (was "Balance").
	require.Contains(t, props, "balance")
	assert.NotContains(t, props, "Balance")
	assert.Equal(t, "Balance", props["balance"].Extensions["x-go-name"])

	// swagger:name overriding a json tag → property renamed (was "currency").
	require.Contains(t, props, "currencyCode")
	assert.NotContains(t, props, "currency")
	assert.Equal(t, "Currency", props["currencyCode"].Extensions["x-go-name"])

	// No override → json-tag name retained.
	require.Contains(t, props, "plain")

	scantest.CompareOrDumpJSON(t, doc, "quirk_name_field.json")
}
