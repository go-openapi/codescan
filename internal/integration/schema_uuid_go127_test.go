// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build go1.27

package integration_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

// TestStdlibUUID witnesses end-to-end recognition of the go1.27 stdlib uuid.UUID.
//
// Tagged go1.27 because the fixture cannot compile on an older toolchain ("package uuid is not in
// std"). The recognizer itself is untagged production code — codescan compares types harvested from
// scanned sources and never imports uuid — so the predicate is covered on every supported toolchain
// by TestIsStdUUID in internal/builders/resolvers.
func TestStdlibUUID(t *testing.T) {
	fixturesPath := filepath.Join(scantest.FixturesDir(), "goparsing", "go127", "uuid")
	var (
		sp          *oaispec.Swagger
		diagnostics []codescan.Diagnostic
	)

	t.Run("end-to-end source scan should succeed", func(t *testing.T) {
		var err error
		sp, err = runScan(&codescan.Options{
			WorkDir:    fixturesPath,
			ScanModels: true,
			OnDiagnostic: func(d codescan.Diagnostic) {
				diagnostics = append(diagnostics, d)
			},
		})
		require.NoError(t, err)
	})

	require.NotNil(t, sp)

	order, ok := sp.Definitions["Order"]
	require.TrueT(t, ok)
	props := order.Properties
	require.NotEmpty(t, props)

	assertUUID := func(t *testing.T, s oaispec.Schema) {
		t.Helper()
		assert.TrueT(t, s.Type.Contains("string"))
		assert.EqualT(t, "uuid", s.Format)
	}

	t.Run("a plain uuid.UUID field renders as a formatted string", func(t *testing.T) {
		assertUUID(t, props["id"])
	})

	t.Run("a pointer to uuid.UUID is peeled before recognition", func(t *testing.T) {
		assertUUID(t, props["ptrId"])
	})

	t.Run("a slice of uuid.UUID carries the format on its items", func(t *testing.T) {
		ids := props["ids"]
		require.TrueT(t, ids.Type.Contains("array"))
		require.NotNil(t, ids.Items)
		require.NotNil(t, ids.Items.Schema)
		assertUUID(t, *ids.Items.Schema)
	})

	t.Run("a map value of uuid.UUID carries the format on additionalProperties", func(t *testing.T) {
		byName := props["byName"]
		require.TrueT(t, byName.Type.Contains("object"))
		require.NotNil(t, byName.AdditionalProperties)
		require.NotNil(t, byName.AdditionalProperties.Schema)
		assertUUID(t, *byName.AdditionalProperties.Schema)
	})

	t.Run("uuid.UUID as a map KEY still yields an object", func(t *testing.T) {
		// The key type is not rendered in Swagger 2.0; what matters is that the map is recognized as
		// an object at all, which requires the key to marshal as a JSON string.
		keyed := props["keyed"]
		assert.TrueT(t, keyed.Type.Contains("object"))
	})

	t.Run("an explicit swagger:strfmt still beats the recognizer", func(t *testing.T) {
		overridden := props["overridden"]
		assert.TrueT(t, overridden.Type.Contains("string"))
		assert.EqualT(t, "date", overridden.Format)
	})

	t.Run("a named type over uuid.UUID renders as a formatted string", func(t *testing.T) {
		orderID, ok := sp.Definitions["OrderID"]
		require.TrueT(t, ok)
		assertUUID(t, orderID)
	})

	t.Run("an alias of uuid.UUID dissolves to a formatted string at the use site", func(t *testing.T) {
		tagged, ok := sp.Definitions["Tagged"]
		require.TrueT(t, ok)
		assertUUID(t, tagged.Properties["alias"])
	})

	t.Run("an EMBEDDED uuid.UUID is dropped, with a warning — quirks-open.md Q33", func(t *testing.T) {
		// Current behaviour, not desired behaviour. uuid.UUID has an array underlying type, so
		// buildNamedEmbedded's switch falls to its default arm (only the *interface* arm consults
		// applyStdlibSpecials) and skips the embed. Go itself renders the whole struct as a bare
		// string via the promoted MarshalText, so "name" is not on the wire either.
		//
		// This is a pre-existing, uuid-agnostic quirk shared with every strfmt.UUID / time.Time
		// embed; the identity recognizer deliberately does not change it. When Q33 is decided, this
		// sub-test and the golden move together.
		embedder, ok := sp.Definitions["Embedder"]
		require.TrueT(t, ok)
		assert.TrueT(t, embedder.Type.Contains("object"))
		assert.Len(t, embedder.Properties, 1)
		assert.Contains(t, embedder.Properties, "name")

		var warned bool
		for _, d := range diagnostics {
			if d.Severity == codescan.SeverityWarning && strings.Contains(d.Message, "buildNamedEmbedded") {
				warned = true
				break
			}
		}
		assert.TrueT(t, warned, "the dropped embed must not be silent")
	})

	scantest.CompareOrDumpJSON(t, sp, "go127_uuid_spec.json")
}
