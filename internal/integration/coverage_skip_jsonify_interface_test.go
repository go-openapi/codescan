// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func runSkipJSONifyInterface(t *testing.T, skip bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		Packages:                    []string{"./enhancements/interface-no-mangle/..."},
		WorkDir:                     scantest.FixturesDir(),
		ScanModels:                  true,
		SkipJSONifyInterfaceMethods: skip,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestCoverage_SkipJSONifyInterface_Off is the control: with the flag off the
// default-path methods auto-jsonify (lowercase-first camelCase), matching the
// historic behaviour.
func TestCoverage_SkipJSONifyInterface_Off(t *testing.T) {
	doc := runSkipJSONifyInterface(t, false)

	require.Contains(t, doc.Definitions, "Account")
	props := doc.Definitions["Account"].Properties

	// Default path — mangler runs.
	assert.Contains(t, props, "id", "default path must auto-jsonify when the flag is off")
	assert.Contains(t, props, "createdAt", "default path must auto-jsonify when the flag is off")
	assert.NotContains(t, props, "ID", "the verbatim Go name must not appear when the flag is off")
	assert.NotContains(t, props, "CreatedAt", "the verbatim Go name must not appear when the flag is off")

	// swagger:name override is verbatim regardless.
	assert.Contains(t, props, "explicit_name", "swagger:name override must stay verbatim")
}

// TestCoverage_SkipJSONifyInterface_On exercises Options.SkipJSONifyInterfaceMethods:
// with the opt-out on, default-path interface methods are emitted verbatim (the
// auto-jsonify mangler is skipped) while the swagger:name override still reaches
// the spec exactly as written.
func TestCoverage_SkipJSONifyInterface_On(t *testing.T) {
	doc := runSkipJSONifyInterface(t, true)

	require.Contains(t, doc.Definitions, "Account")
	props := doc.Definitions["Account"].Properties

	// Default path — mangler skipped, Go method name verbatim.
	assert.Contains(t, props, "ID", "opt-out must skip the mangler on the default path")
	assert.Contains(t, props, "CreatedAt", "opt-out must skip the mangler on the default path")
	assert.NotContains(t, props, "id", "the mangled spelling must not appear when opted out")
	assert.NotContains(t, props, "createdAt", "the mangled spelling must not appear when opted out")

	// swagger:name override is honored verbatim regardless of the flag.
	assert.Contains(t, props, "explicit_name", "swagger:name override must stay verbatim")
	assert.NotContains(t, props, "explicitName", "the override must not be re-mangled")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_interface_no_mangle_on.json")
}
