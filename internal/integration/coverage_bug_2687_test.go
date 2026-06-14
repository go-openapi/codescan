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

// TestCoverage_Bug2687 covers go-swagger issue #2687 ("Ignore kubebuilder
// annotations"): tool directive markers (`// +kubebuilder:...`, `// +genclient`,
// `// +k8s:...`) used to leak into the generated model and property
// descriptions. This was also the residual flagged by #3007.
//
// FIXED (fix/go-swagger-2687): the lexer classifies `// +marker`-style lines as
// directives (alongside Go `//go:`/`//nolint:` directives) and drops them from
// the prose surface, so the descriptions stay clean.
func TestCoverage_Bug2687(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2687/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	app := doc.Definitions["Application"]
	// FIXED: the marker lines are stripped; only the authored prose remains.
	assert.Equal(t, "MyType description", app.Description)
	assert.Equal(t, "Name of the type", app.Properties["name"].Description)
	assert.NotContains(t, app.Description, "+kubebuilder")
	assert.NotContains(t, app.Description, "+genclient")
	assert.NotContains(t, app.Properties["name"].Description, "+kubebuilder")

	// The +kubebuilder:validation:Required marker is dropped, but the
	// genuine `required: true` keyword on the same field is still honoured.
	assert.Contains(t, app.Required, "name")
}
