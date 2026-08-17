// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Differential test for embedding a named type whose underlying is neither a struct nor an
// interface: the emitted property set must equal the key set `encoding/json` puts on the wire.
//
// Such an embed promotes nothing — there is no field to promote — so Go keeps the value as an
// ordinary member named after the TYPE. `buildNamedEmbedded` had arms for struct and interface
// only, so every one of these fell to a warn-and-skip default and the member disappeared. The
// warning read "unsupported Go type", which describes a type codescan cannot model rather than one
// it silently drops.
//
// Like json-tag-fidelity this corpus has an ORACLE, so no expectation is written here: the fixture
// module marshals its own types and commits the raw documents as `wire.golden.json`.
//
// # The one deliberate divergence
//
// MarshalHost embeds a type implementing encoding.TextMarshaler, which promotes MarshalText and
// makes the whole struct marshal as a bare string under the DEFAULT marshaller. codescan does not
// model that, by decision: an embed means composition, and a composed model round-trips through a
// hand-written marshaller (as go-swagger's generated models do) rather than the default one. The
// oracle records the divergence rather than hiding it — that is why it stores raw documents rather
// than key sets, since this one is not an object at all.
func TestEmbedBasicUnderlying(t *testing.T) {
	wire := loadEmbedWireGolden(t)

	doc, err := runScan(&codescan.Options{
		Packages:   []string{"./enhancements/embed-basic-underlying/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	for name, raw := range wire {
		t.Run(name, func(t *testing.T) {
			def, ok := doc.Definitions[name]
			require.True(t, ok, "missing definition %s", name)

			got := make([]string, 0, len(def.Properties))
			for k := range def.Properties {
				got = append(got, k)
			}
			sort.Strings(got)

			var obj map[string]json.RawMessage
			if err := json.Unmarshal(raw, &obj); err != nil {
				// Not an object on the wire: the promoted-marshaller case.
				require.Equal(t, "MarshalHost", name,
					"only the promoted-marshaller subject may diverge from the oracle; %s marshalled to %s", name, raw)
				assert.Equal(t, []string{"Token", "label"}, got,
					"the member is built like any other embed of a named type — the promoted marshaller is not modelled")

				return
			}

			want := make([]string, 0, len(obj))
			for k := range obj {
				want = append(want, k)
			}
			sort.Strings(want)

			assert.Equal(t, want, got,
				"the emitted property set must match what encoding/json marshals")
		})
	}

	// The member is BUILT from the embedded type, not merely declared: a classifier on that type
	// reaches it exactly as it would reach any named-type property.
	t.Run("the embedded type's classifiers apply to the member", func(t *testing.T) {
		props := doc.Definitions["FmtHost"].Properties
		assert.Equal(t, "string/duration", schemaSignature(props["FmtBasic"], doc.Definitions, 0))
	})

	t.Run("the embed's json tag names the property", func(t *testing.T) {
		// The one embed shape where the json tag is meaningful again: it names an ordinary property
		// instead of steering a promotion.
		assert.Contains(t, doc.Definitions["TaggedHost"].Properties, "count")
		assert.NotContains(t, doc.Definitions["OmittedHost"].Properties, "Count")
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_embed_basic_underlying.json")
}

// loadEmbedWireGolden reads the raw documents the fixture module captured from encoding/json.
func loadEmbedWireGolden(t *testing.T) map[string]json.RawMessage {
	t.Helper()

	path := filepath.Join(scantest.FixturesDir(), "enhancements", "embed-basic-underlying", "wire.golden.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "wire oracle missing — regenerate with UPDATE_GOLDEN=1 in the fixtures module")

	var wire map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &wire))
	require.NotEmpty(t, wire)

	return wire
}
