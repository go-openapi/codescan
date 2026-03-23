// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package text_marshal exercises schemaBuilder.buildFromTextMarshal:
// a type recognised via its TextMarshaler implementation, the
// case-insensitive "uuid" fast-path, and a named text-marshalable type
// that carries a swagger:strfmt tag.
package text_marshal

// UUID is a text-marshalable type whose name matches the uuid fast-path.
type UUID [16]byte

// MarshalText renders the UUID as text.
func (u UUID) MarshalText() ([]byte, error) { return nil, nil }

// UnmarshalText parses the UUID from text.
func (u *UUID) UnmarshalText([]byte) error { return nil }

// MAC is a text-marshalable named type annotated with a swagger:strfmt so
// the strfmt branch of buildFromTextMarshal fires.
//
// swagger:strfmt mac
type MAC [6]byte

// MarshalText renders the MAC as text.
func (m MAC) MarshalText() ([]byte, error) { return nil, nil }

// UnmarshalText parses the MAC from text.
func (m *MAC) UnmarshalText([]byte) error { return nil }

// Opaque is a text-marshalable named type without any swagger hints so
// that buildFromTextMarshal takes its fallback branch.
type Opaque struct {
	Value string
}

// MarshalText renders the Opaque value as text.
func (o Opaque) MarshalText() ([]byte, error) { return []byte(o.Value), nil }

// UnmarshalText parses the Opaque value from text.
func (o *Opaque) UnmarshalText(data []byte) error { o.Value = string(data); return nil }

// Device aggregates all three text-marshalable fields so that a full scan
// walks buildFromTextMarshal for each.
//
// swagger:model Device
type Device struct {
	// required: true
	ID UUID `json:"id"`

	MAC MAC `json:"mac"`

	Data Opaque `json:"data"`
}
