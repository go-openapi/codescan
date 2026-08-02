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

// --- stdlib-identity subjects: reached as the NAMED type, not through an alias ---
//
// The two subjects above name their stdlib type through an alias, which lands in
// the alias arm. A field typed `time.Time` or `json.RawMessage` DIRECTLY lands in
// the named arm instead, where each builder used to carry its own subset of the
// identity recognizers. Nothing reached that arm before these.
//
// `error` is the one that makes the subsets observable without a truncated
// package graph: it is predeclared, so its object has a nil package and no
// declaration exists to look up in any graph. A builder that demands the
// declaration before consulting the recognizer cannot degrade — it dereferences
// nil.

// ErrAlias names the predeclared error through an alias.
//
// The alias's own object is this name, not `error`, so an identity recognizer
// keyed on the object never fires here — only after the alias dissolves.
type ErrAlias = error

// --- shape subjects: the arms of the field dispatch, rather than the classifiers ---
//
// The subjects above are all named or alias types, which reach only two arms of
// `buildFromField`. These reach the rest — struct, interface, map, slice with an
// inline element, pointer, plain basic — so that a factorization of those arms is
// guarded in all three positions rather than by goldens alone.

// EmailsNamed is a named STRING slice carrying a NON-special format.
//
// This is the pinned divergence. The element-driven rule (see
// common.ApplyArrayLikeStrfmt) asks whether the ELEMENT makes the sequence
// string-like: `byte` and `rune` do, so a format describes the whole value;
// `string` does not, so the format describes each element. The schema builder
// applies that rule. The parameters and responses builders short-circuit on a
// local `strfmtFromDoc` helper that predates it and writes
// `Typed("string", format)` unconditionally, claiming the value IS one email
// when the Go type is a list of them.
//
// swagger:strfmt email
type EmailsNamed []string

// CodesNamed is the array flavour of the same divergence.
//
// swagger:strfmt email
type CodesNamed [4]string

// Plain is a struct reached directly as a field.
type Plain struct {
	// Left is a plain property.
	Left string `json:"left"`
}

// Speaker is a non-empty interface reached directly as a field.
type Speaker interface {
	// Say returns a word.
	Say() string
}

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

	// Struct is the struct-arm subject.
	Struct Plain `json:"struct"`

	// Iface is the interface-arm subject.
	Iface Speaker `json:"iface"`

	// Mapping is the map-arm subject.
	Mapping map[string]Plain `json:"mapping"`

	// Inline is the slice arm with an inline element.
	Inline []struct {
		// Code is the inline element property.
		Code string `json:"code"`
	} `json:"inline"`

	// Ptr is the pointer arm.
	Ptr *Plain `json:"ptr"`

	// Basic is the plain-basic arm.
	Basic int32 `json:"basic"`

	// Emails is the pinned slice+non-special-format divergence.
	Emails EmailsNamed `json:"emails"`

	// Codes is the array flavour of the same.
	Codes CodesNamed `json:"codes"`

	// StampN is the stdlib time reached as the named type.
	StampN time.Time `json:"stampN"`

	// RawN is the open-schema stdlib type reached as the named type.
	RawN json.RawMessage `json:"rawN"`

	// AnyV is the predeclared any.
	AnyV any `json:"anyv"`

	// ErrN is the predeclared error — no package, no declaration.
	ErrN error `json:"errN"`

	// ErrAl names the same through an alias.
	ErrAl ErrAlias `json:"errAl"`
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

// ParamsStruct reaches the struct subject as a body parameter.
//
// swagger:parameters confStruct
type ParamsStruct struct {
	// in: body
	Body Plain `json:"body"`
}

// RespStruct reaches the struct subject as a response body.
//
// swagger:response respStruct
type RespStruct struct {
	// in: body
	Body Plain `json:"body"`
}

// HandlerStruct binds the struct subject.
//
// swagger:route POST /struct conf confStruct
//
// Responses:
//
//	200: respStruct
func HandlerStruct() {}

// ParamsIface reaches the iface subject as a body parameter.
//
// swagger:parameters confIface
type ParamsIface struct {
	// in: body
	Body Speaker `json:"body"`
}

// RespIface reaches the iface subject as a response body.
//
// swagger:response respIface
type RespIface struct {
	// in: body
	Body Speaker `json:"body"`
}

// HandlerIface binds the iface subject.
//
// swagger:route POST /iface conf confIface
//
// Responses:
//
//	200: respIface
func HandlerIface() {}

// ParamsMapping reaches the mapping subject as a body parameter.
//
// swagger:parameters confMapping
type ParamsMapping struct {
	// in: body
	Body map[string]Plain `json:"body"`
}

// RespMapping reaches the mapping subject as a response body.
//
// swagger:response respMapping
type RespMapping struct {
	// in: body
	Body map[string]Plain `json:"body"`
}

// HandlerMapping binds the mapping subject.
//
// swagger:route POST /mapping conf confMapping
//
// Responses:
//
//	200: respMapping
func HandlerMapping() {}

// ParamsPtr reaches the ptr subject as a body parameter.
//
// swagger:parameters confPtr
type ParamsPtr struct {
	// in: body
	Body *Plain `json:"body"`
}

// RespPtr reaches the ptr subject as a response body.
//
// swagger:response respPtr
type RespPtr struct {
	// in: body
	Body *Plain `json:"body"`
}

// HandlerPtr binds the ptr subject.
//
// swagger:route POST /ptr conf confPtr
//
// Responses:
//
//	200: respPtr
func HandlerPtr() {}

// ParamsBasic reaches the basic subject as a body parameter.
//
// swagger:parameters confBasic
type ParamsBasic struct {
	// in: body
	Body int32 `json:"body"`
}

// RespBasic reaches the basic subject as a response body.
//
// swagger:response respBasic
type RespBasic struct {
	// in: body
	Body int32 `json:"body"`
}

// HandlerBasic binds the basic subject.
//
// swagger:route POST /basic conf confBasic
//
// Responses:
//
//	200: respBasic
func HandlerBasic() {}

// ParamsInline reaches the inline-element slice as a body parameter.
//
// swagger:parameters confInline
type ParamsInline struct {
	// in: body
	Body []struct {
		// Code is the inline element property.
		Code string `json:"code"`
	} `json:"body"`
}

// RespInline reaches the inline-element slice as a response body.
//
// swagger:response respInline
type RespInline struct {
	// in: body
	Body []struct {
		// Code is the inline element property.
		Code string `json:"code"`
	} `json:"body"`
}

// HandlerInline binds the inline-slice subject.
//
// swagger:route POST /inline conf confInline
//
// Responses:
//
//	200: respInline
func HandlerInline() {}

// ParamsEmails reaches the emails subject as a body parameter.
//
// swagger:parameters confEmails
type ParamsEmails struct {
	// in: body
	Body EmailsNamed `json:"body"`
}

// RespEmails reaches the emails subject as a response body.
//
// swagger:response respEmails
type RespEmails struct {
	// in: body
	Body EmailsNamed `json:"body"`
}

// HandlerEmails binds the emails subject.
//
// swagger:route POST /emails conf confEmails
//
// Responses:
//
//	200: respEmails
func HandlerEmails() {}

// ParamsCodes reaches the codes subject as a body parameter.
//
// swagger:parameters confCodes
type ParamsCodes struct {
	// in: body
	Body CodesNamed `json:"body"`
}

// RespCodes reaches the codes subject as a response body.
//
// swagger:response respCodes
type RespCodes struct {
	// in: body
	Body CodesNamed `json:"body"`
}

// HandlerCodes binds the codes subject.
//
// swagger:route POST /codes conf confCodes
//
// Responses:
//
//	200: respCodes
func HandlerCodes() {}

// --- stdlib-identity subjects in the other two positions ---

// ParamsStampN reaches the named stdlib time as a body parameter.
//
// swagger:parameters confStampN
type ParamsStampN struct {
	// in: body
	Body time.Time `json:"body"`
}

// RespStampN reaches the named stdlib time as a response body.
//
// swagger:response respStampN
type RespStampN struct {
	// in: body
	Body time.Time `json:"body"`
}

// HandlerStampN binds the named stdlib time subject.
//
// swagger:route POST /stamp-n conf confStampN
//
// Responses:
//
//	200: respStampN
func HandlerStampN() {}

// ParamsRawN reaches the named open-schema type as a body parameter.
//
// swagger:parameters confRawN
type ParamsRawN struct {
	// in: body
	Body json.RawMessage `json:"body"`
}

// RespRawN reaches the named open-schema type as a response body.
//
// swagger:response respRawN
type RespRawN struct {
	// in: body
	Body json.RawMessage `json:"body"`
}

// HandlerRawN binds the named open-schema subject.
//
// swagger:route POST /raw-n conf confRawN
//
// Responses:
//
//	200: respRawN
func HandlerRawN() {}

// ParamsAnyV reaches the predeclared any as a body parameter.
//
// swagger:parameters confAnyV
type ParamsAnyV struct {
	// in: body
	Body any `json:"body"`
}

// RespAnyV reaches the predeclared any as a response body.
//
// swagger:response respAnyV
type RespAnyV struct {
	// in: body
	Body any `json:"body"`
}

// HandlerAnyV binds the predeclared-any subject.
//
// swagger:route POST /anyv conf confAnyV
//
// Responses:
//
//	200: respAnyV
func HandlerAnyV() {}

// ParamsErrN reaches the predeclared error as a body parameter.
//
// swagger:parameters confErrN
type ParamsErrN struct {
	// The name is the subject's, not the usual "body": the diagnostic raised when this parameter is
	// dropped has to be attributable to it.
	//
	// in: body
	Body error `json:"errN"`
}

// RespErrN reaches the predeclared error as a response body.
//
// swagger:response respErrN
type RespErrN struct {
	// in: body
	Body error `json:"body"`
}

// HandlerErrN binds the predeclared-error subject.
//
// swagger:route POST /err-n conf confErrN
//
// Responses:
//
//	200: respErrN
func HandlerErrN() {}

// ParamsErrAl reaches the aliased error as a body parameter.
//
// swagger:parameters confErrAl
type ParamsErrAl struct {
	// Named for the subject, as above.
	//
	// in: body
	Body ErrAlias `json:"errAl"`
}

// RespErrAl reaches the aliased error as a response body.
//
// swagger:response respErrAl
type RespErrAl struct {
	// in: body
	Body ErrAlias `json:"body"`
}

// HandlerErrAl binds the aliased-error subject.
//
// swagger:route POST /err-al conf confErrAl
//
// Responses:
//
//	200: respErrAl
func HandlerErrAl() {}
