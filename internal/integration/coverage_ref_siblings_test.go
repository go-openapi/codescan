// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// A field whose type is a model becomes a $ref, and draft-4 forbids siblings beside a $ref — so a
// keyword written on such a field has to ride an allOf compound instead. One collector decides
// that, and each value shape reaches it by a different arm: number, integer, bool, string, raw
// block, extension.
//
// The existing coverage only ever wrote an extension and an example, so four of those six arms were
// never entered.
func TestCoverage_RefSiblingValidations(t *testing.T) {
	var diags []codescan.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:     []string{"./enhancements/ref-sibling-validations/..."},
		WorkDir:      scantest.FixturesDir(),
		ScanModels:   true,
		OnDiagnostic: func(d codescan.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)

	bag, ok := doc.Definitions["Bag"]
	require.True(t, ok)

	// override returns the compound's second arm — where a validation sibling lands.
	override := func(t *testing.T, field string) spec.Schema {
		t.Helper()
		p, ok := bag.Properties[field]
		require.True(t, ok, "property %q", field)
		require.Len(t, p.AllOf, 2, "%q wraps the $ref and its overrides", field)
		assert.Equal(t, "#/definitions/Item", p.AllOf[0].Ref.String(), "the reference is preserved")

		return p.AllOf[1]
	}

	t.Run("numeric validations", func(t *testing.T) {
		o := override(t, "ranged")
		require.NotNil(t, o.Maximum)
		assert.InDelta(t, 100.0, *o.Maximum, 0)
		require.NotNil(t, o.Minimum)
		assert.InDelta(t, 1.0, *o.Minimum, 0)
		require.NotNil(t, o.MultipleOf)
		assert.InDelta(t, 5.0, *o.MultipleOf, 0)
	})

	t.Run("length and item-count validations", func(t *testing.T) {
		o := override(t, "sized")
		require.NotNil(t, o.MinLength)
		assert.Equal(t, int64(1), *o.MinLength)
		require.NotNil(t, o.MaxLength)
		assert.Equal(t, int64(64), *o.MaxLength)
		require.NotNil(t, o.MinItems)
		assert.Equal(t, int64(2), *o.MinItems)
		require.NotNil(t, o.MaxItems)
		assert.Equal(t, int64(8), *o.MaxItems)
	})

	t.Run("property-count validations", func(t *testing.T) {
		o := override(t, "counted")
		require.NotNil(t, o.MinProperties)
		assert.Equal(t, int64(1), *o.MinProperties)
		require.NotNil(t, o.MaxProperties)
		assert.Equal(t, int64(4), *o.MaxProperties)
	})

	t.Run("boolean validations", func(t *testing.T) {
		o := override(t, "flagged")
		assert.True(t, o.UniqueItems)
		assert.True(t, o.ReadOnly)

		// `required` is not a sibling at all: it belongs to the ENCLOSING object, so it lands on the
		// parent's required list rather than on the compound.
		assert.Contains(t, bag.Required, "flagged")
	})

	t.Run("string-shaped validations", func(t *testing.T) {
		o := override(t, "matched")
		assert.Equal(t, "^[a-z]+$", o.Pattern)
		// The override arm carries no type of its own, so a JSON literal is coerced structurally.
		assert.Equal(t, map[string]any{"name": "fallback"}, o.Default)
		assert.Equal(t, map[string]any{"name": "sample"}, o.Example)
	})

	t.Run("map-shaped validations", func(t *testing.T) {
		o := override(t, "keyed")
		assert.Contains(t, o.PatternProperties, "^x-")
		require.NotNil(t, o.AdditionalProperties)
		assert.Equal(t, spec.StringOrArray{"string"}, o.AdditionalProperties.Schema.Type)
	})

	t.Run("scalar-valued default, example and enum", func(t *testing.T) {
		o := override(t, "scalar")
		assert.Equal(t, "fallback", o.Default)
		assert.Equal(t, "sample", o.Example)
		assert.Equal(t, []any{"alpha", "beta", "gamma"}, o.Enum)
	})

	// externalDocs and extensions ride the OUTER compound rather than the override arm, so the
	// field carries all of its metadata at one level.
	t.Run("externalDocs lifts onto the compound", func(t *testing.T) {
		p := bag.Properties["documented"]
		require.Len(t, p.AllOf, 1, "nothing for an override arm to hold")
		require.NotNil(t, p.ExternalDocs)
		assert.Equal(t, "https://example.com/bag", p.ExternalDocs.URL)
		assert.NotEmpty(t, p.Description, "the description stays on the outer schema")
	})

	t.Run("extensions lift onto the compound", func(t *testing.T) {
		p := bag.Properties["extended"]
		require.Len(t, p.AllOf, 1)
		assert.Equal(t, float64(3), p.Extensions["x-order"])
		assert.Equal(t, true, p.Extensions["x-internal"])
		assert.NotEmpty(t, p.Description)
	})

	// A bare `x-foo:` line is not the extension grammar — it is prose. On a $ref'd field that
	// leaves only a description, and the legacy default emits a bare {$ref} rather than compounding
	// for prose alone (§ref-override). The output is deliberate; what was missing was any word of
	// it, so the drop now raises a Hint naming the option that would keep it.
	t.Run("a description-only $ref reports what it drops", func(t *testing.T) {
		p := bag.Properties["bareExt"]

		assert.Equal(t, "#/definitions/Item", p.Ref.String(), "still a bare $ref")
		assert.Empty(t, p.Description)
		assert.Empty(t, p.Extensions)

		var hint *codescan.Diagnostic
		for i, d := range diags {
			if d.Code == grammar.CodeDroppedRefSibling && strings.Contains(d.Message, "bareExt") {
				hint = &diags[i]

				break
			}
		}
		require.NotNil(t, hint, "the drop must be reported, not silent")
		assert.Equal(t, codescan.SeverityHint, hint.Severity,
			"nothing is wrong — the default simply cannot carry prose beside a $ref")
		assert.Contains(t, hint.Message, "EmitRefSiblings", "and it names the way to keep it")
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_ref_sibling_validations.json")
}
