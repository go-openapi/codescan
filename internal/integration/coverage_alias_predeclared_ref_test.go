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

// TestCoverage_AliasPredeclaredRef regression-tests the nil-pkg panic that fires in
// `buildDeclAlias`'s $ref branch when the alias's RHS is a predeclared type — concretely `type
// Err = error`.
//
// The predeclared `error` interface has no package (universe scope); `ro.Pkg()` returns nil, and
// the GetModel lookup `ro.Pkg().Path()` nil-panics.
//
// The fix in `buildDeclAlias`'s `case *types.Named` branch guards the nil package case and routes
// through `applyStdlibSpecials`, so `error` produces its canonical `{type: string, x-go-type:
// error}` shape inline instead of crashing the scan.
func TestCoverage_AliasPredeclaredRef(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-calibration-stdlib/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		RefAliases: true,
	})
	require.NoError(t, err, "no panic; no error; spec built")
	require.NotNil(t, doc)

	// All four stdlib aliases must produce their canonical inline shape directly on their own
	// definition — no chain hop through a separately-built target.
	// Lifting applyStdlibSpecials above the nil-pkg conditional unified what was a within-mode
	// asymmetry (predeclared `error` inline; packaged stdlib types chained).

	require.Contains(t, doc.Definitions, "Err",
		"swagger:model Err = error must produce an Err definition")
	errDef := doc.Definitions["Err"]
	assert.Equal(t, []string{"string"}, []string(errDef.Type),
		"Err must canonicalise to type:string via recognizeError")
	assert.Equal(t, "error", errDef.Extensions["x-go-type"],
		"Err must carry the x-go-type:error extension")
	assert.Empty(t, errDef.Ref.String(),
		"Err must NOT chain via $ref — recognizer emits inline")

	require.Contains(t, doc.Definitions, "Timestamp")
	ts := doc.Definitions["Timestamp"]
	assert.Equal(t, []string{"string"}, []string(ts.Type))
	assert.Equal(t, "date-time", ts.Format)
	assert.Empty(t, ts.Ref.String(),
		"Timestamp must NOT chain to a separate Time definition under Ref")

	require.Contains(t, doc.Definitions, "Raw")
	raw := doc.Definitions["Raw"]
	assert.Empty(t, raw.Type,
		"Raw is the open-shape recognizer output (matches `any` behaviour)")
	assert.Empty(t, raw.Ref.String(),
		"Raw must NOT chain to a separate RawMessage definition under Ref")

	// Unannotated aliases are a Go implementation detail and do not surface as standalone definitions.
	// SilentTime dissolves at its use site (Envelope.silent); the recognizer still wins for the inline
	// shape.
	assert.NotContains(t, doc.Definitions, "SilentTime",
		"unannotated alias of time.Time must not produce a standalone definition")
	require.Contains(t, doc.Definitions, "Envelope")
	silentField := doc.Definitions["Envelope"].Properties["silent"]
	assert.Equal(t, []string{"string"}, []string(silentField.Type),
		"Envelope.silent inlines recognizeTime's canonical type:string at the use site")
	assert.Equal(t, "date-time", silentField.Format,
		"Envelope.silent inlines recognizeTime's canonical format:date-time at the use site")
	assert.Empty(t, silentField.Ref.String(),
		"Envelope.silent must be inline; no $ref now that SilentTime is dissolved")

	// Side benefit: the chain targets no longer pollute definitions.
	assert.NotContains(t, doc.Definitions, "Time",
		"no Time def — Timestamp/SilentTime inline directly, no chain target")
	assert.NotContains(t, doc.Definitions, "RawMessage",
		"no RawMessage def — Raw inlines the open shape, no chain target")
}

// TestCoverage_AliasStdlibDefault pins the stdlib-recognizer behaviour on `buildDeclAlias`'s Expand
// branch: aliases of stdlib-special types (time.Time, error, json.RawMessage, any) produce their
// canonical shape directly in Default mode instead of the structural-walk wrong shape that
// buildFromType(Underlying) would otherwise emit.
//
// Without the recognizer call before the Underlying fallthrough:
//   - Timestamp = time.Time → {type: object} (walked the empty struct)
//   - Err = error → {type: object} (walked the interface)
//   - Raw = json.RawMessage → {type: array, items: {integer, uint8}}
//   - SilentTime = time.Time → {type: object}
//
// With the recognizer call, all four produce their canonical shape inline, matching
// TransparentAliases mode.
func TestCoverage_AliasStdlibDefault(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-calibration-stdlib/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Timestamp = time.Time → {type: string, format: date-time}
	require.Contains(t, doc.Definitions, "Timestamp")
	ts := doc.Definitions["Timestamp"]
	assert.Equal(t, []string{"string"}, []string(ts.Type),
		"Timestamp must canonicalise to type:string via recognizeTime")
	assert.Equal(t, "date-time", ts.Format,
		"Timestamp must carry format:date-time")

	// Err = error → {type: string, x-go-type: error}
	require.Contains(t, doc.Definitions, "Err")
	errDef := doc.Definitions["Err"]
	assert.Equal(t, []string{"string"}, []string(errDef.Type),
		"Err must canonicalise to type:string via recognizeError")
	assert.Equal(t, "error", errDef.Extensions["x-go-type"],
		"Err must carry the x-go-type:error extension")

	// SilentTime = time.Time (UNANNOTATED, reachable via field) — unannotated aliases do not produce
	// standalone definitions.
	// The recognizer's canonical shape ({string, date-time}) lands inline at the use site
	// (Envelope.silent) instead of on a SilentTime def.
	assert.NotContains(t, doc.Definitions, "SilentTime",
		"unannotated alias of time.Time must not produce a standalone definition under Default")
	require.Contains(t, doc.Definitions, "Envelope")
	silentField := doc.Definitions["Envelope"].Properties["silent"]
	assert.Equal(t, []string{"string"}, []string(silentField.Type),
		"Envelope.silent inlines recognizeTime's canonical type:string at the use site")
	assert.Equal(t, "date-time", silentField.Format)
	assert.Empty(t, silentField.Ref.String(),
		"Envelope.silent must be inline; no $ref now that SilentTime is dissolved")

	// Raw = json.RawMessage → recognizeRawMessage produces the open "any JSON" shape
	// (target.Schema() with no Typed() call).
	// The emitted Raw definition has no `type` keyword — same as `any`.
	// The shape is recognizer-canonical even if visually ambiguous.
	require.Contains(t, doc.Definitions, "Raw")
	raw := doc.Definitions["Raw"]
	assert.Empty(t, raw.Type,
		"Raw is the open-shape recognizer output — no type keyword (matches `any` behaviour)")
}
