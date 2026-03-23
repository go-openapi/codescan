// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package allof_edges exercises allOf composition edge cases: pointer
// members, time.Time members, strfmt-tagged members, and interface members.
package allof_edges

import "time"

// SimpleBase is a plain struct used as an allOf member.
//
// swagger:model
type SimpleBase struct {
	// required: true
	ID string `json:"id"`

	Name string `json:"name"`
}

// ULID is a named struct formatted as a swagger strfmt. Using a struct
// underlying type exercises the strfmt branch of buildNamedAllOf for
// struct members.
//
// swagger:strfmt ulid
type ULID struct{}

// Tagger is a non-empty named interface used as an allOf member.
//
// swagger:model
type Tagger interface {
	// Tag returns an identifier.
	Tag() string
}

// AllOfPointer composes a pointer allOf member.
//
// swagger:model
type AllOfPointer struct {
	// swagger:allOf
	*SimpleBase

	Extra string `json:"extra"`
}

// AllOfStdTime composes an allOf member that is time.Time.
//
// swagger:model
type AllOfStdTime struct {
	// swagger:allOf
	time.Time

	Label string `json:"label"`
}

// AllOfStrfmt composes an allOf member that carries a swagger:strfmt tag.
//
// swagger:model
type AllOfStrfmt struct {
	// swagger:allOf
	ULID

	Note string `json:"note"`
}

// AllOfInterface composes an allOf member whose underlying type is an
// interface.
//
// swagger:model
type AllOfInterface struct {
	// swagger:allOf
	Tagger

	Count int `json:"count"`
}
