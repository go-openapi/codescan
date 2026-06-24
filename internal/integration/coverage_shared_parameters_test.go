// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_SharedParameters_TopLevel exercises P2 of the
// shared-parameters feature (go-swagger#2632): a `swagger:parameters *`
// struct registers its fields at the spec top level (#/parameters/{name}),
// keyed by the resolved parameter name. Register-only: the entries appear
// in the top-level map regardless of whether any operation references them
// yet (the $ref wiring is a later phase).
func TestCoverage_SharedParameters_TopLevel(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/shared-parameters/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// CommonHeaders (swagger:parameters *) → #/parameters/X-Request-ID
	// AuthHeader   (swagger:parameters * createPet) → #/parameters/X-API-Key
	reqID, ok := doc.Parameters["X-Request-ID"]
	require.TrueT(t, ok, "expected #/parameters/X-Request-ID")
	assert.EqualT(t, "header", reqID.In)
	assert.EqualT(t, "X-Request-ID", reqID.Name)
	assert.EqualT(t, "string", reqID.Type)

	apiKey, ok := doc.Parameters["X-API-Key"]
	require.TrueT(t, ok, "expected #/parameters/X-API-Key")
	assert.EqualT(t, "header", apiKey.In)
	assert.TrueT(t, apiKey.Required)

	// The operation-id targets (ListPetsParams → listPets, CreatePetParams →
	// createPet) are NOT registered at the top level.
	_, hasLimit := doc.Parameters["limit"]
	assert.FalseT(t, hasLimit, "inline operation params must not leak into #/parameters")
}

// TestCoverage_SharedParameters_Conflict exercises the keep-first conflict
// policy (P2, fixture 3): two `swagger:parameters *` structs in different
// packages register the same short name. The first wins, the later is
// dropped with a scan.shared-parameter-conflict warning — never renamed,
// since shared parameters are referenced only by short name. Independent
// namespaces: #/definitions/Status coexists with #/parameters/Status.
func TestCoverage_SharedParameters_Conflict(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/shared-parameters-conflict/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// X-Token registered once (the survivor wins; the loser is dropped, not
	// renamed — no X-Token clone under a mangled key). pkga (header) is
	// scanned before pkgb (query), so the header form is the keep-first
	// survivor.
	xtoken, ok := doc.Parameters["X-Token"]
	require.TrueT(t, ok, "expected #/parameters/X-Token")
	assert.EqualT(t, "header", xtoken.In, "pkga (header) wins keep-first over pkgb (query)")
	assert.Len(t, doc.Parameters, 2, "X-Token + Status, no renamed duplicate")

	// Independent namespaces: a shared parameter named Status coexists with a
	// definition named Status.
	_, hasParamStatus := doc.Parameters["Status"]
	assert.TrueT(t, hasParamStatus, "expected #/parameters/Status")
	_, hasDefStatus := doc.Definitions["Status"]
	assert.TrueT(t, hasDefStatus, "expected #/definitions/Status (independent namespace)")

	// A keep-first conflict warning was emitted for X-Token.
	var sawConflict bool
	for _, d := range diags {
		if d.Code == grammar.CodeSharedParameterConflict {
			sawConflict = true
		}
	}
	assert.TrueT(t, sawConflict, "expected a scan.shared-parameter-conflict warning")
}
