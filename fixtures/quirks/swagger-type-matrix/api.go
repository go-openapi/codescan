// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package swaggertypematrix is the F3 witness: it captures the CURRENT
// rendering of swagger:type across its argument vocabulary, in combination
// with swagger:strfmt, and at both the field site and the named-type site —
// BEFORE any rationalisation. No fix is applied; the golden documents today's
// (often surprising, site-dependent) behaviour so the F3 fix is judged
// against a known baseline. See doc-site-quirks.md F3.
package swaggertypematrix

// Custom is a scanned struct type, referenced by the Matrix fields below.
// A field typed Custom whose override is dropped falls through to the Go
// type (an inlined struct, as the golden shows).
//
// swagger:model Custom
type Custom struct {
	X int `json:"x"`
}

// Matrix exercises field-level swagger:type across the argument space. Every
// field's Go type is Custom, so the baseline (no override) would be a $ref.
//
// swagger:model Matrix
type Matrix struct {
	// swagger:type string
	AString Custom `json:"aString"`
	// swagger:type integer
	BInteger Custom `json:"bInteger"`
	// swagger:type int64
	CInt64 Custom `json:"cInt64"`
	// swagger:type number
	DNumber Custom `json:"dNumber"`
	// swagger:type boolean
	EBoolean Custom `json:"eBoolean"`
	// swagger:type object
	FObject Custom `json:"fObject"`
	// swagger:type []string
	HArrayExplicit Custom `json:"hArrayExplicit"`
	// swagger:type file
	IFile Custom `json:"iFile"`
	// swagger:type badValue
	JBad Custom `json:"jBad"`
	// swagger:type Custom
	KScanned Custom `json:"kScanned"`
	// strfmt THEN type (field site) — which wins?
	//
	// swagger:strfmt uuid
	// swagger:type integer
	LStrfmtThenType Custom `json:"lStrfmtThenType"`
	// strfmt THEN string-type (field site).
	//
	// swagger:strfmt uuid
	// swagger:type string
	MStrfmtThenString Custom `json:"mStrfmtThenString"`
	// A genuine Go slice field carrying swagger:type array (Mode-2 idiom).
	//
	// swagger:type array
	GArray []string `json:"gArray"`
	// inline forces the field's own Go type (Custom) to expand in place,
	// rather than the $ref a plain Custom field would emit.
	//
	// swagger:type inline
	NInline Custom `json:"nInline"`
	// []Custom is an array whose items are the inlined Custom type.
	//
	// swagger:type []Custom
	OArrayCustom Custom `json:"oArrayCustom"`
	// A cross-type reference: a string field overridden to the Custom type,
	// inlined (swagger:type always inlines, regardless of the Go type).
	//
	// swagger:type Custom
	PCrossRef string `json:"pCrossRef"`
}

// NamedSlice is a named slice carrying the Mode-2 `array` inline idiom.
//
// swagger:type array
// swagger:model NamedSlice
type NamedSlice []string

// NamedScalarString is a named int overridden to string at the named site.
//
// swagger:type string
// swagger:model NamedScalarString
type NamedScalarString int

// NamedStrfmtAndType carries BOTH strfmt and type at the named-scalar site —
// the named-site precedence may differ from the field site above.
//
// swagger:strfmt uuid
// swagger:type integer
// swagger:model NamedStrfmtAndType
type NamedStrfmtAndType int
