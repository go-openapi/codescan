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

// TestCoverage_SingleLineCommentAsDescription locks the answer to go-swagger
// issue #2626: the opt-in SingleLineCommentAsDescription option routes a
// one-line doc comment to the model `description` (never `title`), while a
// multi-line comment keeps the existing title/description split.
func TestCoverage_SingleLineCommentAsDescription(t *testing.T) {
	pkgs := []string{"./enhancements/single-line-description/..."}

	// Default: a single-line comment ending in a period is the title.
	def, err := codescan.Run(&codescan.Options{
		Packages: pkgs, WorkDir: scantest.FixturesDir(), ScanModels: true,
	})
	require.NoError(t, err)
	widget := def.Definitions["Widget"]
	assert.Equal(t, "Widget is a one-line model comment ending in a period.", widget.Title)
	assert.Empty(t, widget.Description)

	// Option on: the single-line comment becomes the description instead.
	on, err := codescan.Run(&codescan.Options{
		Packages: pkgs, WorkDir: scantest.FixturesDir(), ScanModels: true,
		SingleLineCommentAsDescription: true,
	})
	require.NoError(t, err)
	widgetOn := on.Definitions["Widget"]
	assert.Empty(t, widgetOn.Title, "single-line comment no longer a title (go-swagger#2626)")
	assert.Equal(t, "Widget is a one-line model comment ending in a period.", widgetOn.Description)

	// Multi-line comments are unaffected by the option: the title/description
	// split is preserved in both modes.
	gadgetDef := def.Definitions["Gadget"]
	gadgetOn := on.Definitions["Gadget"]
	assert.Equal(t, "Gadget is the title line.", gadgetDef.Title)
	assert.Equal(t, "And this is the multi-line description body.", gadgetDef.Description)
	assert.Equal(t, gadgetDef.Title, gadgetOn.Title, "multi-line title unchanged by the option")
	assert.Equal(t, gadgetDef.Description, gadgetOn.Description, "multi-line description unchanged by the option")
}
