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

// TestCoverage_Bug301 locks the resolution to go-swagger issue #301
// ("swagger:model huh?"): a swagger:model is emitted with its full title /
// description split and field validations. The reporter's "model not appearing"
// was the -m (ScanModels) requirement — an unreferenced model only shows up
// under -m, which is asserted here as the documented behaviour.
func TestCoverage_Bug301(t *testing.T) {
	withModels, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/301/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	user, ok := withModels.Definitions["User"]
	require.True(t, ok, "the swagger:model appears under -m")

	assert.Equal(t, "User represents the user for this application", user.Title)
	assert.Contains(t, user.Description, "security principal")
	assert.ElementsMatch(t, []string{"id", "name", "login"}, user.Required)
	assert.Equal(t, float64(1), *user.Properties["id"].Minimum)
	assert.Equal(t, int64(3), *user.Properties["name"].MinLength)
	assert.Equal(t, "email", user.Properties["login"].Format)
	assert.Equal(t, "#/definitions/User",
		user.Properties["friends"].Items.Schema.Ref.String(), "self-referential array")

	// The documented confusion: without -m an unreferenced model is absent.
	withoutModels, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/301/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	assert.NotContains(t, withoutModels.Definitions, "User",
		"an unreferenced swagger:model is omitted without -m (the #301 'huh?')")

	scantest.CompareOrDumpJSON(t, withModels, "bugs_301_schema.json")
}
