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

// TestCoverage_Bug2655 verifies the fix for go-swagger issue #2655: a `Tags:`
// block in swagger:meta now populates the top-level `tags` section
// (spec.Swagger.Tags) instead of being swallowed into info.description. Each
// tag carries its name, description, optional externalDocs and vendor
// extensions — the YAML list nesting survives the raw-block lexer because
// `Tags:` preserves per-line indentation (like extensions/securityDefinitions).
func TestCoverage_Bug2655(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2655/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// The Tags block now lands on doc.Tags, no longer leaking its content
	// into the info.description.
	require.NotNil(t, doc.Info)
	assert.False(t, strings.Contains(doc.Info.Description, "Everything about your Pets"),
		"the Tags block content must not leak into info.description")

	require.Len(t, doc.Tags, 2)

	pet := doc.Tags[0]
	assert.Equal(t, "pet", pet.Name)
	assert.Equal(t, "Everything about your Pets", pet.Description)
	require.NotNil(t, pet.ExternalDocs, "nested externalDocs must survive")
	assert.Equal(t, "Find out more", pet.ExternalDocs.Description)
	assert.Equal(t, "http://swagger.io", pet.ExternalDocs.URL)

	store := doc.Tags[1]
	assert.Equal(t, "store", store.Name)
	assert.Equal(t, "Access to Petstore orders", store.Description)
	assert.Equal(t, "Store", store.Extensions["x-display-name"],
		"per-tag vendor extension must survive")

	// Operation-level tags are a plain string list (not objects). A
	// swagger:route unions its header-line tag (`pets`) with the body
	// `Tags:` keyword (`pets`, `store`), deduping to [pets, store]. A
	// swagger:operation gets the same list straight from its wholesale
	// YAML body.
	require.NotNil(t, doc.Paths)
	listPets := doc.Paths.Paths["/pets"].Get
	require.NotNil(t, listPets, "GET /pets must be present")
	assert.Equal(t, []string{"pets", "store"}, listPets.Tags,
		"route header + Tags: keyword must union and dedup")

	getPet := doc.Paths.Paths["/pets/{id}"].Get
	require.NotNil(t, getPet, "GET /pets/{id} must be present")
	assert.Equal(t, []string{"pets", "store"}, getPet.Tags,
		"swagger:operation YAML-body tags must land as a string list")
}
