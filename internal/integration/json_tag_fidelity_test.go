// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Differential test for `json:"-"` handling: the emitted property set must equal the key set
// `encoding/json` actually puts on the wire.
//
// This corpus is unusual in having an ORACLE — the right answer is not a design choice but whatever
// encoding/json does. The expectations are therefore not written here: the fixture module marshals
// its own types and commits the resulting key sets as `wire.golden.json` (see that package's
// wire_test.go for why the two sides meet at a file rather than an import), and this test compares
// against them.
//
// Two shapes used to diverge:
//
//   - a promoted field re-declared with `json:"-"` was deleted, though Go ignores such a field
//     entirely — it never shadows the promoted one, which Go still marshals;
//   - `json:"-,"`, the escape for a field literally named `-`, was dropped rather than emitted.
func TestJSONTagFidelity(t *testing.T) {
	wire := loadWireGolden(t)

	var diags []string
	doc, err := codescan.Run(&codescan.Options{
		Packages:     []string{"./enhancements/json-tag-fidelity/..."},
		WorkDir:      scantest.FixturesDir(),
		ScanModels:   true,
		OnDiagnostic: func(d codescan.Diagnostic) { diags = append(diags, d.String()) },
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	for name, wantKeys := range wire {
		t.Run(name, func(t *testing.T) {
			def, ok := doc.Definitions[name]
			require.True(t, ok, "missing definition %s", name)

			got := make([]string, 0, len(def.Properties))
			for k := range def.Properties {
				got = append(got, k)
			}
			sort.Strings(got)

			assert.Equal(t, wantKeys, got,
				"the emitted property set must match what encoding/json marshals")
		})
	}

	// The Hint stays: an author writing `json:"-"` over a promoted field usually means "drop it",
	// which the schema no longer does for them. swagger:omit is the honest way to say it.
	t.Run("shadowed embed still hints", func(t *testing.T) {
		var hinted bool
		for _, d := range diags {
			if strings.Contains(d, "scan.shadowed-embed-field") {
				hinted = true

				break
			}
		}
		assert.True(t, hinted, "re-declaring a promoted field with json:\"-\" must still raise the Hint; got %v", diags)
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_json_tag_fidelity.json")
}

// loadWireGolden reads the key sets the fixture module captured from encoding/json.
func loadWireGolden(t *testing.T) map[string][]string {
	t.Helper()

	path := filepath.Join(scantest.FixturesDir(), "enhancements", "json-tag-fidelity", "wire.golden.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "wire oracle missing — regenerate with UPDATE_GOLDEN=1 in the fixtures module")

	var wire map[string][]string
	require.NoError(t, json.Unmarshal(data, &wire))
	require.NotEmpty(t, wire)

	return wire
}
