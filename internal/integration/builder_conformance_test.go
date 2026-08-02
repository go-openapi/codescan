// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Cross-builder conformance: the schema, parameters and responses builders must agree.
//
// Each of the three resolves Go types to spec constructs and each carries its own copy of rules the
// others also need. Nothing forces them to agree and nothing detected it when they stopped, so a fix
// verified on one read as complete — which is how `swagger:type` on an alias came to work for a
// model field and a query parameter while silently dropping for a body parameter. This test is the
// detector.
//
// It compares three FULL-SCHEMA positions, where no legitimate difference exists:
//
//	a model field  ·  a body parameter  ·  a response body
//
// SimpleSchema positions are excluded on purpose. A non-body parameter and a response header have a
// genuinely different legality surface — `type` mandatory and restricted, `$ref` forbidden — which
// is the historical reason the builders grew separate paths at all. Comparing them needs a declared
// projection rather than equality, and mixing the two would bury real drift under expected
// difference.
//
// The subjects carry no hand-written expectations: each is asserted against the model field, whose
// behaviour is pinned by its own witnesses elsewhere. This suite only asks whether the three agree.
func TestBuilderConformance(t *testing.T) {
	var diags []codescan.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/builder-conformance/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d codescan.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	subjects := []struct {
		prop     string // property on ModelHost
		path     string // route carrying the body parameter
		response string // response whose body carries it
		note     string

		// paramDropped marks a subject the parameters builder must REFUSE rather than render.
		// The reason is location-specific and stated, not emergent: it is the only shape in this
		// suite where a full-schema position legitimately produces nothing.
		paramDropped string
	}{
		{prop: "fmt", path: "/fmt", response: "respFmt"},
		{prop: "fmtAl", path: "/fmt-al", response: "respFmtAl"},
		{prop: "typ", path: "/typ", response: "respTyp"},
		{
			prop: "typAl", path: "/typ-al", response: "respTypAl",
			note: "the pair that caught the body-branch gap: swagger:type on an alias",
		},
		{prop: "enum", path: "/enum", response: "respEnum"},
		{prop: "bytes", path: "/bytes", response: "respBytes"},
		{prop: "stamp", path: "/stamp", response: "respStamp"},
		{prop: "raw", path: "/raw", response: "respRaw"},

		// Shape subjects: the arms of the field dispatch rather than the classifiers. The classifier
		// subjects above reach only the Named and Alias arms; these reach the rest, so a factorization
		// of those arms is guarded in all three positions.
		{prop: "struct", path: "/struct", response: "respStruct"},
		{prop: "iface", path: "/iface", response: "respIface"},
		{prop: "mapping", path: "/mapping", response: "respMapping"},
		{prop: "inline", path: "/inline", response: "respInline", note: "slice arm with an inline element"},
		{prop: "ptr", path: "/ptr", response: "respPtr"},
		{prop: "basic", path: "/basic", response: "respBasic"},

		{
			prop: "emails", path: "/emails", response: "respEmails",
			note: "named []string + non-special format — the element-driven rule puts it on items",
		},
		{prop: "codes", path: "/codes", response: "respCodes", note: "array flavour of the same"},

		// Stdlib-identity subjects. The classifier subjects above name their stdlib type through an
		// alias; these reach it as the NAMED type, which is the arm where each builder carried its own
		// subset of the recognizers.
		{prop: "stampN", path: "/stamp-n", response: "respStampN", note: "time.Time as the named type"},
		{prop: "rawN", path: "/raw-n", response: "respRawN", note: "json.RawMessage as the named type"},
		{prop: "anyv", path: "/anyv", response: "respAnyV", note: "the predeclared any"},
		{
			prop: "errN", path: "/err-n", response: "respErrN",
			note:         "the predeclared error: nil package, so no declaration exists to look up",
			paramDropped: "an error has no meaning as something a client sends",
		},
		{
			prop: "errAl", path: "/err-al", response: "respErrAl",
			note:         "the same through an alias — the recognizer only fires after the dissolve",
			paramDropped: "an error has no meaning as something a client sends",
		},
	}

	// Cells where the builders are legitimately expected to disagree. Empty: the three full-schema
	// positions have no reason to differ, so every divergence found here has been a defect.
	//
	// The assertion runs in BOTH directions, so a listed cell that starts agreeing fails too and the
	// list cannot rot into a stale TODO.
	knownBroken := map[string]string{}

	model := doc.Definitions["ModelHost"].Properties
	require.NotEmpty(t, model, "the control host must have properties")

	var ledger strings.Builder
	fmt.Fprintf(&ledger, "\n%-8s %-26s %-26s %-26s %s\n",
		"SUBJECT", "MODEL FIELD", "BODY PARAM", "RESPONSE BODY", "")
	fmt.Fprintf(&ledger, "%s\n", strings.Repeat("-", 100))

	for _, s := range subjects {
		t.Run(s.prop, func(t *testing.T) {
			want, ok := model[s.prop]
			require.True(t, ok, "missing control property %s", s.prop)

			control := schemaSignature(want, doc.Definitions, 0)
			asParam, hasParam := bodyParamSignature(t, doc, s.path)
			asResponse := responseBodySignature(t, doc, s.response)

			if s.paramDropped != "" {
				// The refusal must be reported, not silent: a parameter vanishing from an operation with no
				// word to the author is the failure mode the skip-with-a-diagnostic rule exists to avoid.
				assert.False(t, hasParam,
					"%s: the parameters builder must drop this (%s), but it emitted %s",
					s.prop, s.paramDropped, asParam)
				assert.True(t, hasDiagnosticFor(diags, s.prop),
					"%s: dropped without a diagnostic naming it", s.prop)
				assert.Equal(t, control, asResponse,
					"a dropped parameter says nothing about the response, which must still agree")

				fmt.Fprintf(&ledger, "%-8s %-26s %-26s %-26s %s\n",
					s.prop, control, "<dropped>", asResponse, "DECLARED — "+s.paramDropped)

				return
			}
			require.True(t, hasParam, "%s: no body parameter on %s", s.prop, s.path)

			paramAgrees := control == asParam
			responseAgrees := control == asResponse
			agrees := paramAgrees && responseAgrees
			reason, pinned := knownBroken[s.prop]

			var verdict string
			switch {
			case agrees && pinned:
				verdict = "UNPINNED — remove it from knownBroken"
			case agrees:
				verdict = "OK"
			case pinned:
				verdict = "PINNED(Q39)"
			default:
				verdict = "DIVERGES"
			}
			if s.note != "" {
				verdict += " — " + s.note
			}
			fmt.Fprintf(&ledger, "%-8s %-26s %-26s %-26s %s\n", s.prop, control, asParam, asResponse, verdict)

			if pinned {
				assert.False(t, agrees,
					"%s: pinned as broken (%s) but the builders now agree — remove it", s.prop, reason)

				return
			}
			assert.Equal(t, control, asParam,
				"the parameters builder must render this shape as the schema builder does")
			assert.Equal(t, control, asResponse,
				"the responses builder must render this shape as the schema builder does")
		})
	}

	t.Log(ledger.String())

	// The comparison above only asks whether the three builders agree; a wrong answer they all share
	// would pass it. The golden makes every subject's emitted spec reviewable on its own.
	scantest.CompareOrDumpJSON(t, doc, "enhancements_builder_conformance.json")
}

// bodyParamSignature renders the body parameter of the operation on path, reporting whether one was
// emitted at all. A missing parameter is a result rather than a fatality: a subject the builder is
// required to refuse has to be distinguishable from one it silently lost.
func bodyParamSignature(t *testing.T, doc *oaispec.Swagger, path string) (string, bool) {
	t.Helper()

	require.NotNil(t, doc.Paths, "fixture must produce paths")
	item, ok := doc.Paths.Paths[path]
	require.True(t, ok, "missing path %s", path)
	require.NotNil(t, item.Post, "missing POST on %s", path)

	for _, p := range item.Post.Parameters {
		if p.In != "body" {
			continue
		}
		require.NotNil(t, p.Schema, "body parameter on %s carries no schema", path)

		return schemaSignature(*p.Schema, doc.Definitions, 0), true
	}

	return "", false
}

// hasDiagnosticFor reports whether any diagnostic names the subject's property, which is also the
// Go field name the fixture gives it in every position.
func hasDiagnosticFor(diags []codescan.Diagnostic, prop string) bool {
	for _, d := range diags {
		if strings.Contains(strings.ToLower(d.Message), strings.ToLower(prop)) {
			return true
		}
	}

	return false
}

// responseBodySignature renders the body schema of the named response.
func responseBodySignature(t *testing.T, doc *oaispec.Swagger, name string) string {
	t.Helper()

	resp, ok := doc.Responses[name]
	require.True(t, ok, "missing response %s", name)
	require.NotNil(t, resp.Schema, "response %s carries no body schema", name)

	return schemaSignature(*resp.Schema, doc.Definitions, 0)
}
