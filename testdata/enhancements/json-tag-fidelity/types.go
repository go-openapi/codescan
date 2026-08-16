// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package json_tag_fidelity witnesses that the emitted property set matches what
// `encoding/json` actually puts on the wire, for every shape of the `json:"-"`
// tag.
//
// This corpus has an ORACLE, which is unusual: the correct answer is not a design
// choice, it is whatever encoding/json does. `wire_test.go` marshals each type
// here and captures the resulting key set as `wire.golden.json`; the integration
// test scans the same types and asserts the property sets agree. Neither side
// hard-codes an expectation.
//
// Values are chosen non-zero so `omitempty` never hides a key that the wire would
// otherwise carry.
package json_tag_fidelity

// Base is the shared embedded type. Every promoted name below comes from here.
type Base struct {
	// ID is a plain promoted field.
	ID int64 `json:"id"`

	// Name is a plain promoted field.
	Name string `json:"name"`

	// Age is the field the outer structs re-declare.
	Age int32 `json:"age"`
}

// IgnoreShadow re-declares a promoted field with `json:"-"`.
//
// encoding/json ignores a `-` field ENTIRELY: it never enters the name set, so it
// does not shadow the promoted `age`, which Go still marshals. An author writing
// this usually means "drop it" — `swagger:omit` on the embed is the honest way to
// say that, and the scan raises a Hint pointing there.
//
// swagger:model IgnoreShadow
type IgnoreShadow struct {
	Base

	Age int32 `json:"-"`
}

// RenameShadow re-declares a promoted field under a real name — the control.
// Here Go's depth rule DOES apply and the outer declaration wins.
//
// swagger:model RenameShadow
type RenameShadow struct {
	Base

	Age int32 `json:"age"`
}

// PlainIgnore carries `json:"-"` with nothing to shadow — the control for the
// common case, where dropping the property is correct.
//
// swagger:model PlainIgnore
type PlainIgnore struct {
	// Keep stays on the wire.
	Keep string `json:"keep"`

	// Drop is ignored entirely by encoding/json.
	Drop string `json:"-"`
}

// DashName uses the `json:"-,"` escape, which names the field literally `-`
// rather than ignoring it. The trailing comma is the whole difference.
//
// swagger:model DashName
type DashName struct {
	// Weird is emitted under the name "-".
	Weird string `json:"-,"`
}

// DashNameOmitEmpty is the `-,omitempty` variant, which the historic corpus
// already contains (classification/models/nomodel.go). Non-zero here, so the key
// is present on the wire.
//
// swagger:model DashNameOmitEmpty
type DashNameOmitEmpty struct {
	// Weird is emitted under the name "-" when non-empty.
	Weird string `json:"-,omitempty"`
}

// EmbedIgnored tags the EMBED itself `json:"-"`, which does drop the whole embed
// — the control showing that `-` on an embed and `-` on a re-declaration are
// different acts.
//
// swagger:model EmbedIgnored
type EmbedIgnored struct {
	Base `json:"-"`

	// Extra is the only field that survives.
	Extra string `json:"extra"`
}
