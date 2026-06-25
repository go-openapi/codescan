// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package godoclink

import (
	"testing"

	"github.com/go-openapi/swag/mangling"
	"github.com/go-openapi/testify/v2/assert"
)

// fakeResolver maps a doc-link reference to a Resolution from a fixed table —
// standing in for the real go/types resolver (wired in a later phase).
func fakeResolver(table map[string]Resolution) Resolver {
	return func(ref string) (Resolution, bool) {
		r, ok := table[ref]
		return r, ok
	}
}

func TestClean_EmitsMarkersForResolvableLinks(t *testing.T) {
	m := mangling.NewNameMangler()
	resolve := fakeResolver(map[string]Resolution{
		"Person":         {DefKey: "x/models/Person"},
		"Order.CustName": {DefKey: "x/models/Order", Suffix: ".customer_name"},
	})

	out := Clean("Owned by a [Person]; see [Order.CustName] and [Unknown].",
		Options{Mangler: &m, Resolver: resolve})

	// resolvable links became markers carrying the definition key.
	assert.True(t, HasMarkers(out))
	assert.Contains(t, out, encodeMarker("x/models/Person", "", "person", false))
	assert.Contains(t, out, encodeMarker("x/models/Order", ".customer_name", "cust name", false))
	// the unresolved link is humanized in place (no marker).
	assert.Contains(t, out, "unknown")
	assert.NotContains(t, out, "[Unknown]")
}

func TestClean_LeadingSelfNameMarker(t *testing.T) {
	m := mangling.NewNameMangler()
	out := Clean("Widget is owned by a [Person].",
		Options{
			Mangler:  &m,
			Resolver: fakeResolver(map[string]Resolution{"Person": {DefKey: "x/Person"}}),
			Self:     &SelfRef{Name: "Widget", DefKey: "x/Widget"},
		})

	// the leading self-name became a titleizing marker for its own def key.
	assert.Contains(t, out, encodeMarker("x/Widget", "", "widget", true))
}

func TestClean_NoSelfMarkerWithoutResolver(t *testing.T) {
	m := mangling.NewNameMangler()
	// With no Resolver, the leading self-name is left verbatim (its exposed name
	// is unknowable) and links are humanized — the P1 / live behaviour.
	out := Clean("Widget is owned by a [Person].",
		Options{Mangler: &m, Self: &SelfRef{Name: "Widget", DefKey: "x/Widget"}})

	assert.False(t, HasMarkers(out))
	assert.Equal(t, "Widget is owned by a person.", out)
}

func TestSubstituteMarkers_RoundTrip(t *testing.T) {
	m := mangling.NewNameMangler()
	resolve := fakeResolver(map[string]Resolution{
		"Person":         {DefKey: "x/Person"},
		"Order.CustName": {DefKey: "x/Order", Suffix: ".customer_name"},
		"Gone":           {DefKey: "x/Gone"},
	})
	emitted := Clean("Widget owns a [Person], a [Order.CustName] and a [Gone] thing.",
		Options{
			Mangler:  &m,
			Resolver: resolve,
			Self:     &SelfRef{Name: "Widget", DefKey: "x/Widget"},
		})

	// final names: Widget renamed (gizmo), Person/Order kept exposed lowercase,
	// Gone pruned (not an emitted definition) → falls back to its humanized leaf.
	final := map[string]string{
		"x/Widget": "gizmo",
		"x/Person": "person",
		"x/Order":  "order",
	}
	got := SubstituteMarkers(emitted, func(k string) (string, bool) {
		n, ok := final[k]
		return n, ok
	})

	assert.False(t, HasMarkers(got))
	// self-name titleized to the renamed exposed name; field chain joined; pruned
	// key collapsed to fallback.
	assert.Equal(t, "Gizmo owns a person, a order.customer_name and a gone thing.", got)
}

func TestSubstituteMarkers_NoMarkersIsIdentity(t *testing.T) {
	in := "Plain prose with no markers at all."
	assert.Equal(t, in, SubstituteMarkers(in, func(string) (string, bool) { return "", false }))
}
