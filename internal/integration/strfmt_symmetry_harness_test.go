// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Shared harness for the `swagger:strfmt` dispatch-symmetry matrix (Q32).
//
// Each cell is a PAIR of declarations differing only by `=`. The ledger makes two independent
// checks per cell:
//
//   - SYMMETRY — does the alias half agree with its named half? The named half is the control, so
//     the symmetry check never hand-writes a format.
//   - CONTROL  — is the named half itself right? Without this, a dispatch site where BOTH halves
//     ignore the annotation would report a comfortable "OK". `wantNamed` supplies the expected
//     signature; leave it empty to skip the check.
//
// The two checks have separate exception lists, because they are separate defects: `knownBroken`
// is Q32 (alias diverges from named), `controlBroken` is a shared gap (neither half honours the
// annotation). Both fail in BOTH directions — an entry that starts passing is an error too — so
// neither list can rot into a stale TODO.
//
// The four matrix slices live in strfmt_symmetry_{core,composition,simpleschema,stdlib}_test.go.

// symmetryCell is one named/alias pair observed at one dispatch site.
//
// When definition is set, named/alias name two PROPERTIES of it — the use-site cells, where the
// pair is reached from a field. When definition is empty, they name two DEFINITIONS — the
// composition cells (embed, allOf), where the pair is the composing type itself and there is no
// enclosing property to look at.
type symmetryCell struct {
	definition string // definition carrying both properties; "" ⇒ named/alias are definition names
	namedProp  string // property (or definition) reaching the named half
	aliasProp  string // property (or definition) reaching the alias half
	wantNamed  string // expected signature of the named control; "" skips the control check
	note       string // free-text appended to the ledger verdict; for truths the checks cannot assert

	// signatures overrides the definition/property lookup for locations that are not schemas at all
	// — parameters and response headers carry SimpleSchema, not a spec.Schema. When set, definition
	// is ignored and namedProp/aliasProp serve only as the cell's label.
	signatures func(t *testing.T, doc *oaispec.Swagger) (named, alias string)
}

// resolve returns the two schemas a cell compares.
func (c symmetryCell) resolve(t *testing.T, defs oaispec.Definitions) (named, alias oaispec.Schema) {
	t.Helper()

	if c.definition == "" {
		namedDef, okNamed := defs[c.namedProp]
		require.True(t, okNamed, "missing definition %s", c.namedProp)
		aliasDef, okAlias := defs[c.aliasProp]
		require.True(t, okAlias, "missing definition %s", c.aliasProp)
		return namedDef, aliasDef
	}

	def, ok := defs[c.definition]
	require.True(t, ok, "missing definition %s", c.definition)

	namedProp, okNamed := def.Properties[c.namedProp]
	require.True(t, okNamed, "missing property %s.%s", c.definition, c.namedProp)
	aliasProp, okAlias := def.Properties[c.aliasProp]
	require.True(t, okAlias, "missing property %s.%s", c.definition, c.aliasProp)

	return namedProp, aliasProp
}

// symmetryLedger is one fixture package's matrix, run across all three alias modes.
type symmetryLedger struct {
	pkg          string // package pattern under fixtures/
	goldenPrefix string // golden file stem; the mode name is appended
	cells        []symmetryCell

	// exceptions are cells where named and alias SHOULD differ, keyed "<mode>/<cell>".
	exceptions map[string]string
	// knownBroken are cells asymmetric because of Q32, keyed "<mode>/<cell>". The fix's worklist.
	knownBroken map[string]string
	// controlBroken are cells whose NAMED half is already wrong, keyed "<mode>/<cell>" — a shared
	// gap in the dispatch, not an alias problem.
	controlBroken map[string]string
}

// aliasModes are the three alias-handling modes every matrix runs under.
func aliasModes() []struct {
	name        string
	refAliases  bool
	transparent bool
} {
	return []struct {
		name        string
		refAliases  bool
		transparent bool
	}{
		{"default", false, false},
		{"refaliases", true, false},
		{"transparentaliases", false, true},
	}
}

// schemaSignature renders a schema's observable shape as a comparable string, resolving `$ref`
// through defs so an inlined schema and a referenced one compare equal when they describe the same
// thing. Depth-bounded: a cyclic $ref yields "<cycle>" rather than hanging.
func schemaSignature(s oaispec.Schema, defs oaispec.Definitions, depth int) string {
	const maxDepth = 8
	if depth > maxDepth {
		return "<cycle>"
	}

	if ref := s.Ref.String(); ref != "" {
		name := strings.TrimPrefix(ref, "#/definitions/")
		target, ok := defs[name]
		if !ok {
			return "<dangling:" + name + ">"
		}
		return schemaSignature(target, defs, depth+1)
	}

	if len(s.AllOf) > 0 {
		parts := make([]string, 0, len(s.AllOf))
		for _, member := range s.AllOf {
			parts = append(parts, schemaSignature(member, defs, depth+1))
		}
		return "allOf[" + strings.Join(parts, "+") + "]"
	}

	if s.Items != nil && s.Items.Schema != nil {
		return "array<" + schemaSignature(*s.Items.Schema, defs, depth+1) + ">"
	}

	if s.AdditionalProperties != nil && s.AdditionalProperties.Schema != nil {
		return "map<" + schemaSignature(*s.AdditionalProperties.Schema, defs, depth+1) + ">"
	}

	if len(s.Properties) > 0 {
		keys := make([]string, 0, len(s.Properties))
		for k := range s.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return "object{" + strings.Join(keys, ",") + "}"
	}

	typ := ""
	if len(s.Type) > 0 {
		typ = strings.Join(s.Type, "|")
	}
	if typ == "" && s.Format == "" {
		return "<empty>"
	}
	return typ + "/" + s.Format
}

// run scans the fixture package in each alias mode, renders the ledger, and asserts both checks.
func (l symmetryLedger) run(t *testing.T) {
	t.Helper()

	for _, mode := range aliasModes() {
		t.Run(mode.name, func(t *testing.T) {
			doc, err := codescan.Run(&codescan.Options{
				Packages:           []string{"./" + l.pkg + "/..."},
				WorkDir:            scantest.FixturesDir(),
				ScanModels:         true,
				RefAliases:         mode.refAliases,
				TransparentAliases: mode.transparent,
			})
			require.NoError(t, err)
			require.NotNil(t, doc)

			var ledger strings.Builder
			fmt.Fprintf(&ledger, "\n%-24s %-34s %-34s %s\n", "CELL", "NAMED (control)", "ALIAS", "VERDICT")
			fmt.Fprintf(&ledger, "%s\n", strings.Repeat("-", 120))

			for _, c := range l.cells {
				var namedSig, aliasSig string
				if c.signatures != nil {
					namedSig, aliasSig = c.signatures(t, doc)
				} else {
					namedSchema, aliasSchema := c.resolve(t, doc.Definitions)
					namedSig = schemaSignature(namedSchema, doc.Definitions, 0)
					aliasSig = schemaSignature(aliasSchema, doc.Definitions, 0)
				}

				cellID := strings.TrimSuffix(c.namedProp, "Named")
				key := mode.name + "/" + cellID

				verdict := l.verdict(key, namedSig, aliasSig, c.wantNamed)
				if c.note != "" {
					verdict += " — " + c.note
				}
				fmt.Fprintf(&ledger, "%-24s %-34s %-34s %s\n", cellID, namedSig, aliasSig, verdict)

				l.checkControl(t, key, c.wantNamed, namedSig)
				l.checkSymmetry(t, key, namedSig, aliasSig)
			}

			t.Log(ledger.String())

			scantest.CompareOrDumpJSON(t, doc, l.goldenPrefix+"_"+mode.name+".json")
		})
	}
}

// verdict renders the per-cell marker shown in the ledger.
func (l symmetryLedger) verdict(key, namedSig, aliasSig, wantNamed string) string {
	controlWrong := wantNamed != "" && namedSig != wantNamed

	switch {
	case namedSig == aliasSig && controlWrong:
		return "SYMMETRIC but control wrong (want " + wantNamed + ")"
	case namedSig == aliasSig:
		return "OK"
	case l.exceptions[key] != "":
		return "EXPECTED-DIFF"
	case l.knownBroken[key] != "":
		return "BROKEN(Q32)"
	default:
		return "UNEXPECTED"
	}
}

// checkControl asserts the named half against its expected signature, honouring controlBroken.
func (l symmetryLedger) checkControl(t *testing.T, key, wantNamed, namedSig string) {
	t.Helper()

	if wantNamed == "" {
		return
	}
	if reason, isBroken := l.controlBroken[key]; isBroken {
		assert.NotEqual(t, wantNamed, namedSig,
			"%s: control listed as broken (%s) but is now correct — remove it from controlBroken", key, reason)
		return
	}
	assert.Equal(t, wantNamed, namedSig, "%s: the named control itself is wrong", key)
}

// checkSymmetry asserts the alias half against its named control, honouring exceptions/knownBroken.
func (l symmetryLedger) checkSymmetry(t *testing.T, key, namedSig, aliasSig string) {
	t.Helper()

	if reason, isException := l.exceptions[key]; isException {
		assert.NotEqual(t, namedSig, aliasSig,
			"%s: listed as a legitimate difference (%s) but the halves now agree — drop the exception", key, reason)
		return
	}
	if reason, isBroken := l.knownBroken[key]; isBroken {
		assert.NotEqual(t, namedSig, aliasSig,
			"%s: listed as known-broken (%s) but now agrees — remove it from knownBroken", key, reason)
		return
	}
	assert.Equal(t, namedSig, aliasSig, "%s: alias half must match its named control", key)
}

// simpleSignature renders a SimpleSchema location (parameter, response header) in the same
// vocabulary as schemaSignature, so both kinds of cell read alike in the ledger.
func simpleSignature(typ, format string, items *oaispec.Items) string {
	if items != nil {
		return "array<" + simpleSignature(items.Type, items.Format, items.Items) + ">"
	}
	if typ == "" && format == "" {
		return "<empty>"
	}
	return typ + "/" + format
}

// forEveryMode assigns reason to each named cell in all three alias modes, the common case when a
// defect is mode-independent.
func forEveryMode(reason string, cells ...string) map[string]string {
	modes := aliasModes()
	out := make(map[string]string, len(cells)*len(modes))
	for _, mode := range modes {
		for _, cell := range cells {
			out[mode.name+"/"+cell] = reason
		}
	}

	return out
}
