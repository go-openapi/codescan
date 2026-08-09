// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package integration_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// What a dependency SAYS and what it DECLARES are different questions, and a load that takes dependency types from
// the compiler has to answer both.
//
// The first is what the marker scan is for: a `swagger:strfmt` mark lives in the dependency's own source or nowhere,
// so the load reads back the dependencies whose files carry one. The second it cannot anticipate. Any dependency,
// annotated or not, may declare a type that the scanned code goes on to name as a model — and a definition renders
// from its declaration or not at all, so the doc comment, the field tags and the per-field annotations all hang on
// having read source the marker scan had no reason to read.
//
// It used to come out empty. The declaration was fetched on demand, but its FIELDS were located by position, and a
// position out of export data does not index into syntax we parsed ourselves. Every field was skipped in silence and
// the model rendered as a bare name.
//
// So these are the cases where the answer has to come from source the load deliberately did not read.
func TestDependencyDeclarations_ReadBackOnDemand(t *testing.T) {
	t.Parallel()

	// The model is declared in scan-repo-boundary/makeplans, which carries no annotation anywhere in its source: the
	// marker scan passes over the whole package, and the scanned fixture then uses one of its types as a response
	// body. Every property below therefore comes from a file the load had already decided not to open.
	t.Run("a model declared in a dependency that says nothing about itself", func(t *testing.T) {
		t.Parallel()

		doc, notices := scanBookings(t, true)

		booking, ok := doc.Definitions["Booking"]
		require.True(t, ok, "the definition is the declaration; without it the property is a bare name")

		assert.Equal(t, "A Booking in the system", booking.Description,
			"the doc comment above the type, which only source carries")
		assert.Equal(t, []string{"id", "Subject"}, booking.Required,
			"`required: true` written against each field — a field-level annotation, so the field bridge is what reaches it")

		id, ok := booking.Properties["id"]
		require.True(t, ok, "named from the json tag, which is also only in source")
		assert.True(t, id.ReadOnly, "`read only: true`, same field, same comment block")
		assert.Equal(t, "ID the id of the booking", id.Description)

		assert.Contains(t, booking.Properties, "Subject",
			"and a field with no json tag keeps its Go name")

		assert.Empty(t, notices,
			"nothing was missed, so nothing is announced: the notice names a declaration that could not be read")
	})

	// Reading the declaration back must produce the same document as never having skipped it. Whole-document
	// equality rather than property assertions, because the failure this guards against is a property quietly going
	// missing somewhere nobody thought to look.
	t.Run("the document equals the one a full-source scan produces", func(t *testing.T) {
		t.Parallel()

		compiled, _ := scanBookings(t, true)
		full, _ := scanBookings(t, false)

		assert.JSONEq(t, marshal(t, full), marshal(t, compiled))
	})

	// The two shapes the position lookup could not bridge on its own, both from the standard library, both reached
	// through the same fallback: an interface's methods become properties, and an embedded field promotes what it
	// carries. reflect.Type is the first, reflect.Value the second.
	t.Run("interface methods and embedded fields bridge too", func(t *testing.T) {
		t.Parallel()

		compiled, _ := scanTarget(t, "./goparsing/go123/...", true)
		full, _ := scanTarget(t, "./goparsing/go123/...", false)

		rtype, ok := compiled.Definitions["Type"]
		require.True(t, ok)
		assert.NotEmpty(t, rtype.Properties,
			"reflect.Type is an interface: its properties are its methods, each located by name")

		assert.JSONEq(t, marshal(t, full), marshal(t, compiled))
	})
}

// The notice that names an unreadable declaration has to stay rare enough to mean something.
//
// It used to fire on this scan — a lookup for time.Time, costing the spec nothing, since the type is recognised
// from its identity and the declaration was never needed. Silence here is not the absence of a check; it is every
// declaration the builders asked for having been there to read.
//
// What still produces the notice is source that genuinely cannot be reached, which is a virtual filesystem or a
// blob standing in for one — see TestExportOnly_ReportedWhereItCosts, which pins all four outcomes.
func TestDependencyDeclarations_NothingWantedWasMissed(t *testing.T) {
	t.Parallel()

	_, notices := scanTarget(t, "./goparsing/petstore/...", true)

	assert.Empty(t, notices,
		"strfmt is read back for its own marks, time.Time is recognised from its identity, and anything else is "+
			"fetched on demand")
}

func scanBookings(tb testing.TB, compiled bool) (*oaispec.Swagger, []string) {
	tb.Helper()

	return scanTarget(tb, "./goparsing/bookings/...", compiled)
}

// scanTarget runs one fixture bundle, collecting the notices raised where a declaration was wanted and could not be
// read.
func scanTarget(tb testing.TB, target string, compiled bool) (*oaispec.Swagger, []string) {
	tb.Helper()

	var notices []string
	doc, err := codescan.Run(&codescan.Options{
		Packages:             []string{target},
		WorkDir:              scantest.FixturesDir(),
		ScanModels:           true,
		CompiledDependencies: compiled,
		OnDiagnostic: func(d codescan.Diagnostic) {
			if strings.Contains(d.Message, "could not be read") || d.Code == "scan.sourceless-type" {
				notices = append(notices, d.Message)
			}
		},
	})
	require.NoError(tb, err, "one unreadable declaration must not cost the whole document")

	return doc, notices
}

func marshal(tb testing.TB, doc *oaispec.Swagger) string {
	tb.Helper()

	b, err := json.Marshal(doc)
	require.NoError(tb, err)

	return string(b)
}
