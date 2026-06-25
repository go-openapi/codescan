// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"slices"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/spec"
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

// paramRefs collects the #/parameters/{name} $ref targets on an operation.
func paramRefs(op *spec.Operation) []string {
	var refs []string
	for _, p := range op.Parameters {
		if r := p.Ref.String(); r != "" {
			refs = append(refs, r)
		}
	}
	return refs
}

// TestCoverage_SharedParameters_Refs exercises P3 (go-swagger#2632): a
// shared parameter is wired into an operation as a #/parameters/{name}
// $ref through both reference channels — the `swagger:parameters * opid`
// definition convenience and the standalone `swagger:parameters opid name`
// reference marker on a func.
func TestCoverage_SharedParameters_Refs(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/shared-parameters/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotNil(t, doc.Paths)

	pets, ok := doc.Paths.Paths["/pets"]
	require.TrueT(t, ok, "expected a /pets path")

	// createPet: AuthHeader (`swagger:parameters * createPet`) refs X-API-Key.
	require.NotNil(t, pets.Post)
	assert.SliceContainsT(t, paramRefs(pets.Post), "#/parameters/X-API-Key",
		"createPet should $ref the shared X-API-Key (via `* createPet`)")

	// listPets: standalone `swagger:parameters listPets X-Request-ID` ref.
	require.NotNil(t, pets.Get)
	assert.SliceContainsT(t, paramRefs(pets.Get), "#/parameters/X-Request-ID",
		"listPets should $ref the shared X-Request-ID (standalone reference)")
	// the inline query parameter `limit` is still present alongside the ref.
	var hasLimit bool
	for _, p := range pets.Get.Parameters {
		if p.Name == "limit" {
			hasLimit = true
		}
	}
	assert.TrueT(t, hasLimit, "listPets keeps its inline `limit` parameter")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_shared_parameters.json")
}

// TestCoverage_SharedParameters_PathItem exercises P4 (fixture 2,
// go-swagger#2632): `swagger:parameters /path` inlines a struct's fields
// into the path-item; `swagger:parameters /path name` adds a
// #/parameters/{name} $ref to it. Application is exact-path (no hierarchy),
// and path-item parameters co-exist with operation-level ones (the
// operation one wins at resolution — co-presence, not removal).
func TestCoverage_SharedParameters_PathItem(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/shared-parameters-pathitem/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotNil(t, doc.Paths)

	pets, ok := doc.Paths.Paths["/pets"]
	require.TrueT(t, ok, "expected a /pets path")

	// Path-item parameters: inline X-API-Key (from the /pets struct) + a
	// $ref to the shared X-Request-ID (from the /pets reference marker).
	var inlineAPIKey *spec.Parameter
	var refReqID bool
	for i := range pets.Parameters {
		p := pets.Parameters[i]
		switch {
		case p.Name == "X-API-Key" && p.In == "header":
			inlineAPIKey = &pets.Parameters[i]
		case p.Ref.String() == "#/parameters/X-Request-ID":
			refReqID = true
		}
	}
	require.NotNil(t, inlineAPIKey, "expected inline X-API-Key on the /pets path-item")
	assert.TrueT(t, inlineAPIKey.Required, "path-item X-API-Key is required:true")
	assert.TrueT(t, refReqID, "expected a #/parameters/X-Request-ID $ref on the /pets path-item")

	// Co-presence override: listPets carries its OWN X-API-Key (required:false)
	// at the operation level; the path-item's required:true one is untouched.
	require.NotNil(t, pets.Get)
	var opAPIKey *spec.Parameter
	for i := range pets.Get.Parameters {
		if pets.Get.Parameters[i].Name == "X-API-Key" {
			opAPIKey = &pets.Get.Parameters[i]
		}
	}
	require.NotNil(t, opAPIKey, "listPets has its own X-API-Key (operation-level override)")
	assert.FalseT(t, opAPIKey.Required, "operation-level X-API-Key is required:false")

	// Exact path, no hierarchy: /pets/{id} must NOT inherit the /pets
	// path-item parameters.
	petByID, ok := doc.Paths.Paths["/pets/{id}"]
	require.TrueT(t, ok, "expected a /pets/{id} path")
	for _, p := range petByID.Parameters {
		assert.FalseT(t, p.Name == "X-API-Key" || p.Ref.String() == "#/parameters/X-Request-ID",
			"/pets/{id} must not inherit /pets path-item parameters (no hierarchy)")
	}

	// Full-spec snapshot: the path-item parameters array, the #/parameters
	// $ref target, and the co-present operation override are the most novel
	// output of the feature — pin them in a golden for review.
	scantest.CompareOrDumpJSON(t, doc, "enhancements_shared_parameters_pathitem.json")
}

// TestCoverage_SharedResponses exercises P5 (go-swagger#2632): a
// `swagger:response *` struct registers a shared response at #/responses/
// (the `*` is a synonym for the bare/named form, keyed by the type name),
// and operations that name it in their Responses block resolve to a
// $ref: #/responses/{name}. Before P5 the `*` failed the name regex and the
// response was silently dropped, leaving the route ref dangling.
func TestCoverage_SharedResponses(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/shared-parameters/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// ErrorResponse (swagger:response *) is registered at #/responses.
	_, ok := doc.Responses["ErrorResponse"]
	require.TrueT(t, ok, "expected #/responses/ErrorResponse from swagger:response *")

	// Both routes' `default: ErrorResponse` now resolve to a $ref (previously
	// dropped as dangling).
	require.NotNil(t, doc.Paths)
	pets := doc.Paths.Paths["/pets"]
	for _, op := range []*spec.Operation{pets.Get, pets.Post} {
		require.NotNil(t, op)
		require.NotNil(t, op.Responses)
		require.NotNil(t, op.Responses.Default, "operation should have a default response")
		assert.EqualT(t, "#/responses/ErrorResponse", op.Responses.Default.Ref.String())
	}
}

// TestCoverage_SharedParameters_OverridesAndDedup exercises P3 reference
// edge cases (fixture 5, go-swagger#2632): the shared key/reference is the
// resolved (overridden) name (C3); duplicate operation-id targets (C1) and
// duplicate reference names (C2) are dropped with warnings; and a reference
// to an unregistered shared parameter is dropped with a
// scan.dangling-parameter-ref warning.
func TestCoverage_SharedParameters_OverridesAndDedup(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/shared-parameters-overrides/..."},
		WorkDir:  scantest.FixturesDir(),
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// C3: the `name:` override is the registered key (X-Correlation-ID, not
	// the json-tag X-Request-ID), and a reference resolves by that name.
	_, ok := doc.Parameters["X-Correlation-ID"]
	require.TrueT(t, ok, "expected #/parameters/X-Correlation-ID (overridden name)")
	_, hasOld := doc.Parameters["X-Request-ID"]
	assert.FalseT(t, hasOld, "the json-tag name must not be registered when `name:` overrides it")

	require.NotNil(t, doc.Paths)
	things := doc.Paths.Paths["/things"]
	require.NotNil(t, things.Get)
	listRefs := paramRefs(things.Get)
	assert.SliceContainsT(t, listRefs, "#/parameters/X-Correlation-ID",
		"listThings references the shared param by its overridden name")
	// C2: the duplicated X-Correlation-ID reference yields a single $ref.
	var n int
	for _, r := range listRefs {
		if r == "#/parameters/X-Correlation-ID" {
			n++
		}
	}
	assert.EqualT(t, 1, n, "duplicate reference name must collapse to one $ref")
	// dangling: NoSuchParam was dropped, never emitted as a $ref.
	assert.FalseT(t, slices.Contains(listRefs, "#/parameters/NoSuchParam"), "dangling ref must be dropped")

	codes := map[grammar.Code]bool{}
	for _, d := range diags {
		codes[d.Code] = true
	}
	assert.TrueT(t, codes[grammar.CodeDuplicateTarget], "expected a duplicate-target warning (C1)")
	assert.TrueT(t, codes[grammar.CodeDuplicateRef], "expected a duplicate-ref warning (C2)")
	assert.TrueT(t, codes[grammar.CodeDanglingParameterRef], "expected a dangling-parameter-ref warning")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_shared_parameters_overrides.json")
}

// TestCoverage_SharedParameters_YAMLRefs exercises P6 (fixture 4,
// go-swagger#2632): a swagger:operation wholesale-YAML body that references
// the shared namespace is validated against the completed #/parameters and
// #/responses maps. A resolving $ref is kept verbatim; a dangling one is
// dropped with a scan.dangling-{parameter,response}-ref warning rather than
// emitting an invalid reference.
func TestCoverage_SharedParameters_YAMLRefs(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/shared-parameters-yaml/..."},
		WorkDir:  scantest.FixturesDir(),
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotNil(t, doc.Paths)

	// opA: both refs resolve and are kept.
	opA := doc.Paths.Paths["/a"].Get
	require.NotNil(t, opA)
	assert.SliceContainsT(t, paramRefs(opA), "#/parameters/X-Request-ID",
		"opA keeps its resolving #/parameters/X-Request-ID ref")
	require.NotNil(t, opA.Responses)
	require.NotNil(t, opA.Responses.Default)
	assert.EqualT(t, "#/responses/ErrorResponse", opA.Responses.Default.Ref.String())

	// opB: both refs are dangling → dropped.
	opB := doc.Paths.Paths["/b"].Get
	require.NotNil(t, opB)
	assert.FalseT(t, slices.Contains(paramRefs(opB), "#/parameters/DoesNotExist"),
		"opB dangling parameter ref must be dropped")
	if opB.Responses != nil && opB.Responses.Default != nil {
		assert.NotEqualT(t, "#/responses/Missing", opB.Responses.Default.Ref.String(),
			"opB dangling response ref must be dropped")
	}

	codes := map[grammar.Code]bool{}
	for _, d := range diags {
		codes[d.Code] = true
	}
	assert.TrueT(t, codes[grammar.CodeDanglingParameterRef], "expected a dangling-parameter-ref warning")
	assert.TrueT(t, codes[grammar.CodeDanglingResponseRef], "expected a dangling-response-ref warning")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_shared_parameters_yaml.json")
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

	// Responses follow the same keep-first policy: pkga and pkgb both declare
	// `swagger:response *` ErrorResponse; pkga (scanned first by import-path
	// order) wins, pkgb is dropped — registered once, not renamed.
	_, hasErr := doc.Responses["ErrorResponse"]
	require.TrueT(t, hasErr, "expected #/responses/ErrorResponse")
	assert.Len(t, doc.Responses, 1, "ErrorResponse registered once, no renamed duplicate")

	// Keep-first conflict warnings were emitted for both namespaces.
	codes := map[grammar.Code]bool{}
	for _, d := range diags {
		codes[d.Code] = true
	}
	assert.TrueT(t, codes[grammar.CodeSharedParameterConflict], "expected a scan.shared-parameter-conflict warning")
	assert.TrueT(t, codes[grammar.CodeSharedResponseConflict], "expected a scan.shared-response-conflict warning")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_shared_parameters_conflict.json")
}

// runSharedPrune scans Fixture 6 (shared-parameters-prune) under ScanModels,
// toggling PruneUnusedModels, and collects the scan.pruned-unused Hints.
func runSharedPrune(t *testing.T, prune bool) (*spec.Swagger, []grammar.Diagnostic) {
	t.Helper()
	var pruned []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:          []string{"./enhancements/shared-parameters-prune/..."},
		WorkDir:           scantest.FixturesDir(),
		ScanModels:        true,
		PruneUnusedModels: prune,
		OnDiagnostic: func(d grammar.Diagnostic) {
			if d.Code == grammar.CodePrunedUnused {
				pruned = append(pruned, d)
			}
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc, pruned
}

// TestCoverage_SharedParameters_Prune_Off is the control for the prune
// extension (C4, P7): with ScanModels and no PruneUnusedModels, every shared
// parameter and response is emitted — including the ones no operation
// references (X-Unused, UnusedResponse).
func TestCoverage_SharedParameters_Prune_Off(t *testing.T) {
	doc, pruned := runSharedPrune(t, false)

	for _, name := range []string{"X-Used", "X-Unused"} {
		assert.Contains(t, doc.Parameters, name)
	}
	for _, name := range []string{"UsedResponse", "UnusedResponse"} {
		assert.Contains(t, doc.Responses, name)
	}
	assert.Empty(t, pruned, "nothing is pruned without the flag")
}

// TestCoverage_SharedParameters_Prune_On is the core case (C4, P7): ScanModels
// + PruneUnusedModels drops the shared parameter and response that no operation
// or path-item references (X-Unused, UnusedResponse) while keeping the
// referenced pair (X-Used via the standalone reference, UsedResponse via the
// route's Responses block). Each drop raises one located scan.pruned-unused Hint.
func TestCoverage_SharedParameters_Prune_On(t *testing.T) {
	doc, pruned := runSharedPrune(t, true)

	// Referenced shared objects survive.
	assert.Contains(t, doc.Parameters, "X-Used")
	assert.Contains(t, doc.Responses, "UsedResponse")
	assert.Len(t, doc.Parameters, 1, "only the referenced shared parameter survives")
	assert.Len(t, doc.Responses, 1, "only the referenced shared response survives")

	// Unreferenced shared objects are pruned.
	assert.NotContains(t, doc.Parameters, "X-Unused")
	assert.NotContains(t, doc.Responses, "UnusedResponse")

	// The surviving reference still resolves: listP keeps its #/parameters/X-Used
	// $ref and its default #/responses/UsedResponse.
	op := doc.Paths.Paths["/p"].Get
	require.NotNil(t, op)
	assert.Contains(t, paramRefs(op), "#/parameters/X-Used")
	require.NotNil(t, op.Responses.Default)
	assert.EqualT(t, "#/responses/UsedResponse", op.Responses.Default.Ref.String())

	// One located Hint per pruned shared object (severity Hint, sourced line).
	require.Len(t, pruned, 2, "one Hint per pruned shared object")
	for _, d := range pruned {
		assert.Equal(t, grammar.SeverityHint, d.Severity, "a prune is informational")
		assert.Positive(t, d.Pos.Line, "the Hint is located at the pruned declaration")
	}

	scantest.CompareOrDumpJSON(t, doc, "enhancements_shared_parameters_prune.json")
}
