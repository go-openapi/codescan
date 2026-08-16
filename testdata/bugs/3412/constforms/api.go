// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package constforms exercises the const shapes a `swagger:enum` member can take beyond a plain
// literal — the shapes that are invisible to a reader working on literal syntax alone, and which
// the go-swagger#3412 investigation surfaced one after the other:
//
//   - `iota`, where only the first spec of the block carries a type and a value at all and every
//     following spec inherits both implicitly;
//   - constant expressions and references to earlier members (`1 << 3`, `LevelLow * 2`);
//   - rune literals, whose constant is an integer (`'a'` is 97, not the three characters `'a'`);
//   - `true` / `false`, which are predeclared identifiers rather than literals — Go has no boolean
//     literal token;
//   - raw and escaped strings, whose delimiters and escape sequences are part of the syntax, not of
//     the value.
//
// Values come from the type-checker, which has already evaluated every one of these exactly.
package constforms

import "github.com/go-openapi/codescan/testdata/bugs/3412/constforms/foreign"

// Weekday is an iota enum.
//
// swagger:enum Weekday
type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
)

// ForeignDay is declared HERE but typed with an imported type that happens to be called Weekday
// too. Membership is decided by type identity, not by name, so it is not a member of the enum
// above.
const ForeignDay foreign.Weekday = 13

// Level is built from a constant expression and a reference to an earlier member.
//
// swagger:enum Level
type Level int

const (
	LevelLow  Level = 1 << 3
	LevelHigh Level = LevelLow * 2
)

// Letter is a rune enum.
//
// swagger:enum Letter
type Letter rune

const (
	LetterA   Letter = 'a'
	LetterTab Letter = '\t'
)

// Toggle is a bool enum.
//
// swagger:enum Toggle
type Toggle bool

const (
	ToggleOn  Toggle = true
	ToggleOff Toggle = false
)

// Label is a string enum in the raw and escaped literal forms.
//
// swagger:enum Label
type Label string

const (
	LabelRaw     Label = `raw`
	LabelEscaped Label = "a\tb"
)

// Byte is a byte enum, whose members are written as rune literals.
//
// swagger:enum Byte
type Byte byte

const (
	ByteNUL Byte = 0
	ByteX   Byte = 'x'
)

// Settings carries one property per const form.
//
// swagger:model Settings
type Settings struct {
	// The day of the week.
	Weekday Weekday `json:"weekday"`

	// The level.
	Level Level `json:"level"`

	// The letter.
	Letter Letter `json:"letter"`

	// The toggle.
	Toggle Toggle `json:"toggle"`

	// The label.
	Label Label `json:"label"`

	// The byte.
	Byte Byte `json:"byte"`
}
