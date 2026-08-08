// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// scanSourceless runs a target with dependency types taken from the compiler, collecting the
// warnings raised where the document came out thinner for want of a declaration.
func scanSourceless(tb testing.TB, target string, compiled bool) (*oaispec.Swagger, []string) {
	tb.Helper()

	var warned []string
	doc, err := codescan.Run(&codescan.Options{
		Packages:             []string{target},
		WorkDir:              scantest.FixturesDir(),
		ScanModels:           true,
		CompiledDependencies: compiled,
		OnDiagnostic: func(d codescan.Diagnostic) {
			if d.Code != "scan.sourceless-type" {
				return
			}
			assert.Equal(tb, codescan.SeverityWarning, d.Severity,
				"a thinner document is a warning: the scan succeeded and said less than it could have")
			warned = append(warned, d.Message)
		},
	})
	require.NoError(tb, err, "one unreadable declaration must not cost the whole document")

	return doc, warned
}

// A type whose declaring package arrived without source used to take the whole document with it.
//
// It takes a conjunction to reach — the package has to arrive without source, one of its types has
// to be consumed in the emitted surface, and no identity recognizer may claim it first — so what
// lands here is a standard-library type with no obvious wire form. Those are exactly the types that
// must NOT gain a recognizer: a recognizer asserts a wire format for every use of a type, and
// whether an API means nanoseconds, seconds or ISO 8601 by a time.Duration is the author's decision.
// That is what strfmt.Duration is for.
//
// So the scan renders what the type is, says so once, and carries on.
func TestSourcelessType_DegradesInsteadOfFailing(t *testing.T) {
	t.Parallel()

	t.Run("a stdlib type with no wire form of its own", func(t *testing.T) {
		t.Parallel()

		_, warned := scanSourceless(t, "./enhancements/opaque-streams/...", true)

		require.Len(t, warned, 1, "said once, for the type actually consumed")
		assert.Contains(t, warned[0], "io.Writer")
		assert.Contains(t, warned[0], "swagger:description",
			"the warning carries the remedy, since the author is the only one who can supply it")
	})

	t.Run("a type the recognizers answer for warns about nothing", func(t *testing.T) {
		t.Parallel()

		_, warned := scanSourceless(t, "./goparsing/petstore/...", true)

		assert.Empty(t, warned,
			"strfmt.DateTime is read back from source and time.Time is recognized from its identity, "+
				"so nothing here is thinner than it would have been")
	})

	// The heavier loss, and the one that used to pass in silence: not a thinner schema but a definition
	// that is not there at all. The model is declared in a dependency carrying no annotation, so the
	// marker scan never reads it and the property comes out bare, holding its name and nothing else.
	//
	// Only the arm with nothing to fall back on asks for this warning. Where the type can still be
	// rendered structurally the definition is not lost, and a miss there is the ordinary "not a model,
	// inline it" outcome that fires on every scan — which is why this stays silent on the corpus at
	// large rather than firing for every stdlib struct.
	t.Run("a definition that is lost outright, not merely thinner", func(t *testing.T) {
		t.Parallel()

		doc, warned := scanSourceless(t, "./goparsing/bookings/...", true)

		require.Len(t, warned, 1)
		assert.Contains(t, warned[0], "makeplans.Booking")

		assert.NotContains(t, doc.Definitions, "Booking",
			"the warning is the only place this shows up: the document just stops mentioning it")

		resp, ok := doc.Responses["BookingResponse"]
		require.True(t, ok)
		booking, ok := resp.Schema.Properties["booking"]
		require.True(t, ok)
		assert.Empty(t, booking.Ref.String(), "the $ref an ordinary scan emits here is gone")
	})

	// The case that closed outright, and the reason to trust the fallback: a response header typed
	// time.Duration comes out integer/int64, which is what the FULL-source scan produces too. The
	// declaration was only ever going to add prose to it.
	t.Run("the fallback lands on the full-source answer", func(t *testing.T) {
		t.Parallel()

		degraded, warned := scanSourceless(t, "./bugs/2248/...", true)
		require.Len(t, warned, 1)
		assert.Contains(t, warned[0], "time.Duration")

		baseline, baseWarned := scanSourceless(t, "./bugs/2248/...", false)
		assert.Empty(t, baseWarned, "an ordinary scan reads every declaration it asks for")

		header := func(doc *oaispec.Swagger) oaispec.Header {
			t.Helper()

			resp, ok := doc.Responses["indexedNodes"]
			require.True(t, ok)
			h, ok := resp.Headers["lastContact"]
			require.True(t, ok)

			return h
		}
		assert.Equal(t, header(baseline).Type, header(degraded).Type)
		assert.Equal(t, header(baseline).Format, header(degraded).Format,
			"time.Duration renders from its underlying int64 either way")
	})
}
