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

// `swagger:type` on an alias declaration, against the same annotation on a named one — the second
// half of Q32, after `swagger:strfmt`.
//
// `swagger:type` has a wider surface than strfmt (scalar names, `[]T` prefixes, `inline`, and
// references to other scanned types), so each form is its own pair. It also brought a constraint
// strfmt did not: an OAS v2 SimpleSchema location cannot carry every type. See
// TestTypeOverrideSimpleSchema below.
func TestTypeOverrideSymmetry(t *testing.T) {
	ledger := symmetryLedger{
		pkg:          "enhancements/type-override-symmetry",
		goldenPrefix: "type_override_symmetry",
		cells: []symmetryCell{
			{definition: "Envelope", namedProp: "fieldScalarNamed", aliasProp: "fieldScalarAlias", wantNamed: "string/"},
			{definition: "Envelope", namedProp: "fieldArrayNamed", aliasProp: "fieldArrayAlias", wantNamed: "array<string/>"},
			{
				definition: "Envelope", namedProp: "fieldRefNamed", aliasProp: "fieldRefAlias",
				wantNamed: "object{left,right}",
				note:      "a type-name reference inlines the referenced type's shape",
			},
			{
				definition: "Envelope", namedProp: "fieldFormattedNamed", aliasProp: "fieldFormattedAlias",
				wantNamed: "string/uuid",
				note:      "swagger:type wins, a co-present swagger:strfmt rides as an advisory format",
			},
			{definition: "Envelope", namedProp: "pointerScalarNamed", aliasProp: "pointerScalarAlias", wantNamed: "string/"},
			{definition: "Envelope", namedProp: "sliceElemScalarNamed", aliasProp: "sliceElemScalarAlias", wantNamed: "array<string/>"},
			{definition: "Envelope", namedProp: "mapValueScalarNamed", aliasProp: "mapValueScalarAlias", wantNamed: "map<string/>"},

			// The allOf member drops the override on BOTH halves, and the named half emits an empty
			// member. A shared gap rather than an alias asymmetry, so it is deferred to its own quirk
			// rather than folded in here — pinned as a known difference until then.
			{
				namedProp: "AllOfScalarNamed", aliasProp: "AllOfScalarAlias",
				note: "Q40: buildNamedAllOf runs no type classifier — named emits an EMPTY member, alias dissolves",
			},
		},

		exceptions: map[string]string{},
		knownBroken: forEveryMode(
			"Q40 — the allOf member path honours no classifier; deferred, tracked separately",
			"AllOfScalar",
		),
		controlBroken: map[string]string{},
	}

	ledger.run(t)
}

// TestTypeOverrideSimpleSchema pins the constraint `swagger:type` brings that `swagger:strfmt` did
// not: a non-body parameter and a response header are OAS v2 SimpleSchema locations, where `type` is
// mandatory and restricted — an object has no representation there at all.
//
// A legal override applies as anywhere else; an object-resolving one is refused with a diagnostic
// and the Go-derived type stands, because a parameter without a type is not valid. Both halves of
// each pair must agree, which is the point: the gate lives on the shared path, so the alias and the
// named declaration reach it alike.
func TestTypeOverrideSimpleSchema(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/type-override-symmetry/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	op := doc.Paths.Paths["/type-override"].Get
	require.NotNil(t, op)

	params := make(map[string]string, len(op.Parameters))
	for _, p := range op.Parameters {
		params[p.Name] = p.Type
	}
	assert.Equal(t, "string", params["queryScalarNamed"], "a legal override applies to a query parameter")
	assert.Equal(t, params["queryScalarNamed"], params["queryScalarAlias"],
		"the alias half must reach the same classifier as the named half")

	headers := doc.Responses["typeOverrideResponse"].Headers
	assert.Equal(t, "string", headers["X-Named"].Type, "a legal override applies to a response header")
	assert.Equal(t, headers["X-Named"].Type, headers["X-Alias"].Type,
		"the alias half must reach the same classifier as the named half")
}

// TestTypeOverrideFileSynonym pins `swagger:type file` as a synonym for `swagger:file`.
//
// `file` is an OAS v2 type name, so the annotation whose job is naming types should name it;
// `swagger:file` is the older, extraneous spelling and is expected to be deprecated, which makes
// `swagger:type file` the preferred one. Synonymy is implemented by raising the same signal the
// legacy annotation raises, so both spellings pass through the SAME location gate — a formData
// parameter and a response body, nowhere else.
func TestTypeOverrideFileSynonym(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/type-override-symmetry/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	op := doc.Paths.Paths["/file-synonym"].Post
	require.NotNil(t, op)
	byName := make(map[string]string, len(op.Parameters))
	for _, p := range op.Parameters {
		byName[p.Name] = p.Type
	}

	assert.Equal(t, "file", byName["viaAnnotation"], "the legacy spelling still works")
	assert.Equal(t, byName["viaAnnotation"], byName["viaType"],
		"swagger:type file must be identical to swagger:file on a formData parameter")

	// The shared gate means the preferred spelling cannot leak `file` anywhere OAS 2.0 forbids it.
	assert.Equal(t, "string", byName["queryFile"],
		"file is formData-only; elsewhere the override is refused and the Go type stands")

	assert.Equal(t, "file", doc.Responses["fileBodyAnnotation"].Schema.Type[0])
	assert.Equal(t, doc.Responses["fileBodyAnnotation"].Schema.Type[0],
		doc.Responses["fileBodyType"].Schema.Type[0],
		"swagger:type file must be identical to swagger:file on a response body")
}
