// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package math_big witnesses the arbitrary-precision numbers across the positions that can carry
// one, and pins the shape each of them actually takes on the wire.
//
// The three are not interchangeable, which is the whole reason they need a recognizer rather than a
// guess from their names. big.Int carries a MarshalJSON emitting a bare numeric literal, and
// encoding/json prefers json.Marshaler over the MarshalText the type also has — so it travels as a
// JSON number. big.Float and big.Rat have only MarshalText, so they travel quoted: "3.5" for a
// decimal float and "5/3" for a quotient. Unmarshalling enforces the same split; a number offered to
// a *big.Float is rejected, and a string offered to a *big.Int is too.
//
// Before the recognizer, neither half was right. Through a pointer all three satisfied TextMarshaler
// and collapsed onto `string`, which mis-stated big.Int. By value none of them did — the marshal
// methods take a pointer receiver — so they fell to structural drilling and published `Int`, `Float`
// and `Rat` as object definitions carrying math/big's own godoc, the same leak the stream types used
// to produce. The same Go type therefore rendered one way as a pointer and another by value.
//
// Value and pointer agree here on purpose: encoding/json takes the address of an addressable field
// to reach a pointer-receiver marshaller, so a `big.Int` field of a struct marshalled through a
// pointer — what any server does — emits the very same number its `*big.Int` neighbour does.
//
// See [§math-big](../../../internal/builders/schema/README.md#math-big).
package math_big

import "math/big"

// BigModel carries each math/big type twice, once through a pointer and once by value.
//
// Every field is deliberately named UNLIKE its type: the emitted `x-go-name` is the Go FIELD name
// and `x-go-type` is the Go type, and naming a field after its type would make the two extensions
// indistinguishable in the golden and hide a regression in either.
//
// swagger:model BigModel
type BigModel struct {
	Total     *big.Int   `json:"total"`
	Magnitude *big.Float `json:"magnitude"`
	Share     *big.Rat   `json:"share"`

	Count      big.Int   `json:"count"`
	Distance   big.Float `json:"distance"`
	Proportion big.Rat   `json:"proportion"`

	Ledger  []*big.Int          `json:"ledger"`
	Weights map[string]*big.Rat `json:"weights"`
}

// AliasedModel reaches the recognizer through an alias and through a defined type.
//
// Both land on `integer`, by two different routes. An alias IS math/big.Int — the same type object —
// so the recognizer fires on it directly. A defined type is a different type carrying none of
// big.Int's methods, and it is recognized because the scanner keeps the type the declaration was
// *written over*: `type Tally big.Int` means big.Int here for the same reason `type MyTime time.Time`
// has always meant a date-time rather than a drilled struct.
//
// swagger:model AliasedModel
type AliasedModel struct {
	Aliased Amount `json:"aliased"`
	Defined Tally  `json:"defined"`
}

// Amount is an alias, so it resolves to math/big.Int itself.
type Amount = big.Int

// Tally is a defined type over big.Int, carrying none of its methods.
type Tally big.Int

// OverriddenModel is the control: an explicit annotation still wins over the recognizer.
//
// swagger:model OverriddenModel
type OverriddenModel struct {
	// Principal states a wire format, so the recognizer must not overrule it.
	//
	// swagger:strfmt bigdecimal
	Principal *big.Float `json:"principal"`

	// Balance overrides the TYPE rather than the format. A type override replaces the schema
	// outright and the recognizer's x-go-type stamp goes with it, whereas a format override adjusts
	// the format and the stamp survives. Witnessed rather than asserted, because the difference
	// follows from how the two branches are written.
	//
	// swagger:type string
	Balance *big.Int `json:"balance"`
}

// QuoteParams reaches the math/big types from each parameter location.
//
// A non-body parameter builds in SimpleSchema mode, where both answers are legal: `integer` and
// `string` are primitive types, so neither needs a schema.
//
// swagger:parameters getQuote
type QuoteParams struct {
	// Threshold is an arbitrary-precision integer in the query string.
	//
	// in: query
	Threshold *big.Int `json:"threshold"`

	// Tolerance is an arbitrary-precision float in the query string.
	//
	// in: query
	Tolerance *big.Float `json:"tolerance"`

	// Body carries the whole model.
	//
	// in: body
	Body BigModel `json:"body"`
}

// QuoteResponse reaches the math/big types from a response body and a response header.
//
// swagger:response quoteResponse
type QuoteResponse struct {
	// Remaining is an arbitrary-precision count in a header.
	Remaining *big.Int `json:"X-Remaining"`

	// in: body
	Body BigModel `json:"body"`
}
