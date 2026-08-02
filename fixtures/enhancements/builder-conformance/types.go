// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package builder_conformance asserts that the schema, parameters and responses
// builders agree.
//
// Each of the three resolves Go types to spec constructs, and each carries its
// own copy of rules the others also need. Where they diverge, a fix verified on
// one of them reads as complete — which is how `swagger:type` on an alias came to
// work for a model field and a query parameter while silently dropping for a body
// parameter.
//
// # What is compared
//
// One Go shape reached from three FULL-SCHEMA positions, where no legitimate
// difference exists and the three must agree exactly:
//
//   - a model field                (schema builder)
//   - a body parameter             (parameters builder)
//   - a response body              (responses builder)
//
// SimpleSchema positions are deliberately excluded. A non-body parameter and a
// response header have a genuinely different legality surface — `type` is
// mandatory and restricted, `$ref` is forbidden — so they belong in a comparison
// with a declared projection, not in this one. That difference is the historical
// reason the builders grew separate paths at all.
//
// # Why the shapes are NOT nested
//
// Each subject is the field's type DIRECTLY. Nesting them inside a struct sends
// everything through the schema sub-builder by delegation, where the three agree
// trivially and the suite reports a comfortable all-clear. The divergences live in
// the hand-rolled short-circuits that fire when a parameter or response field is
// itself a named or alias type — `buildNamedField` and `buildFieldAlias` — so the
// subject has to be reached that way to be tested at all.
package builder_conformance

import (
	"encoding/json"
	"time"
)

// FmtNamed carries a format on a named type.
//
// swagger:strfmt isbn
type FmtNamed string

// FmtAlias carries the same format on an alias.
//
// swagger:strfmt isbn
type FmtAlias = string

// TypeNamed carries a type override on a named type.
//
// swagger:type string
type TypeNamed int

// TypeAlias carries the same override on an alias. This pair is the one that
// caught the body-branch gap.
//
// swagger:type string
type TypeAlias = int

// EnumNamed is a named enum.
//
// swagger:enum EnumNamed
type EnumNamed uint64

const (
	// EnumLow is the low member.
	EnumLow EnumNamed = 1

	// EnumHigh is the high member.
	EnumHigh EnumNamed = 2
)

// BytesNamed is a byte sequence carrying the whole-schema format.
//
// swagger:strfmt byte
type BytesNamed []byte

// StampAlias is an alias to a recognised stdlib type.
type StampAlias = time.Time

// RawAlias is an alias to the open "any JSON" stdlib type.
type RawAlias = json.RawMessage

// ModelHost reaches every subject as a MODEL FIELD — the schema builder's view,
// and the control for the other two.
//
// swagger:model ModelHost
type ModelHost struct {
	// Fmt is the named-format subject.
	Fmt FmtNamed `json:"fmt"`

	// FmtAl is the alias-format subject.
	FmtAl FmtAlias `json:"fmtAl"`

	// Typ is the named-override subject.
	Typ TypeNamed `json:"typ"`

	// TypAl is the alias-override subject.
	TypAl TypeAlias `json:"typAl"`

	// Enum is the enum subject.
	Enum EnumNamed `json:"enum"`

	// Bytes is the byte-sequence subject.
	Bytes BytesNamed `json:"bytes"`

	// Stamp is the stdlib-alias subject.
	Stamp StampAlias `json:"stamp"`

	// Raw is the open-schema subject.
	Raw RawAlias `json:"raw"`
}

// ParamsFmt reaches FmtNamed as a body parameter.
//
// swagger:parameters confFmt
type ParamsFmt struct {
	// in: body
	Body FmtNamed `json:"body"`
}

// ParamsFmtAl reaches FmtAlias as a body parameter.
//
// swagger:parameters confFmtAl
type ParamsFmtAl struct {
	// in: body
	Body FmtAlias `json:"body"`
}

// ParamsTyp reaches TypeNamed as a body parameter.
//
// swagger:parameters confTyp
type ParamsTyp struct {
	// in: body
	Body TypeNamed `json:"body"`
}

// ParamsTypAl reaches TypeAlias as a body parameter.
//
// swagger:parameters confTypAl
type ParamsTypAl struct {
	// in: body
	Body TypeAlias `json:"body"`
}

// ParamsEnum reaches EnumNamed as a body parameter.
//
// swagger:parameters confEnum
type ParamsEnum struct {
	// in: body
	Body EnumNamed `json:"body"`
}

// ParamsBytes reaches BytesNamed as a body parameter.
//
// swagger:parameters confBytes
type ParamsBytes struct {
	// in: body
	Body BytesNamed `json:"body"`
}

// ParamsStamp reaches StampAlias as a body parameter.
//
// swagger:parameters confStamp
type ParamsStamp struct {
	// in: body
	Body StampAlias `json:"body"`
}

// ParamsRaw reaches RawAlias as a body parameter.
//
// swagger:parameters confRaw
type ParamsRaw struct {
	// in: body
	Body RawAlias `json:"body"`
}

// RespFmt reaches FmtNamed as a response body.
//
// swagger:response respFmt
type RespFmt struct {
	// in: body
	Body FmtNamed `json:"body"`
}

// RespFmtAl reaches FmtAlias as a response body.
//
// swagger:response respFmtAl
type RespFmtAl struct {
	// in: body
	Body FmtAlias `json:"body"`
}

// RespTyp reaches TypeNamed as a response body.
//
// swagger:response respTyp
type RespTyp struct {
	// in: body
	Body TypeNamed `json:"body"`
}

// RespTypAl reaches TypeAlias as a response body.
//
// swagger:response respTypAl
type RespTypAl struct {
	// in: body
	Body TypeAlias `json:"body"`
}

// RespEnum reaches EnumNamed as a response body.
//
// swagger:response respEnum
type RespEnum struct {
	// in: body
	Body EnumNamed `json:"body"`
}

// RespBytes reaches BytesNamed as a response body.
//
// swagger:response respBytes
type RespBytes struct {
	// in: body
	Body BytesNamed `json:"body"`
}

// RespStamp reaches StampAlias as a response body.
//
// swagger:response respStamp
type RespStamp struct {
	// in: body
	Body StampAlias `json:"body"`
}

// RespRaw reaches RawAlias as a response body.
//
// swagger:response respRaw
type RespRaw struct {
	// in: body
	Body RawAlias `json:"body"`
}

// HandlerFmt binds the format subject.
//
// swagger:route POST /fmt conf confFmt
//
// Responses:
//
//	200: respFmt
func HandlerFmt() {}

// HandlerFmtAl binds the alias-format subject.
//
// swagger:route POST /fmt-al conf confFmtAl
//
// Responses:
//
//	200: respFmtAl
func HandlerFmtAl() {}

// HandlerTyp binds the override subject.
//
// swagger:route POST /typ conf confTyp
//
// Responses:
//
//	200: respTyp
func HandlerTyp() {}

// HandlerTypAl binds the alias-override subject.
//
// swagger:route POST /typ-al conf confTypAl
//
// Responses:
//
//	200: respTypAl
func HandlerTypAl() {}

// HandlerEnum binds the enum subject.
//
// swagger:route POST /enum conf confEnum
//
// Responses:
//
//	200: respEnum
func HandlerEnum() {}

// HandlerBytes binds the byte-sequence subject.
//
// swagger:route POST /bytes conf confBytes
//
// Responses:
//
//	200: respBytes
func HandlerBytes() {}

// HandlerStamp binds the stdlib-alias subject.
//
// swagger:route POST /stamp conf confStamp
//
// Responses:
//
//	200: respStamp
func HandlerStamp() {}

// HandlerRaw binds the open-schema subject.
//
// swagger:route POST /raw conf confRaw
//
// Responses:
//
//	200: respRaw
func HandlerRaw() {}
