// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package embedded_types exercises schema-builder edge cases when a struct
// embeds an alias, a named empty interface, or the predeclared error
// interface.
package embedded_types

// Base is a plain struct used as the right-hand side of an alias.
//
// swagger:model
type Base struct {
	// required: true
	ID int64 `json:"id"`

	Name string `json:"name"`
}

// BaseAlias aliases Base so that embedding it exercises the alias branch
// of buildEmbedded.
type BaseAlias = Base

// EmbedsAlias embeds an aliased named struct.
//
// swagger:model
type EmbedsAlias struct {
	BaseAlias

	Extra string `json:"extra"`
}

// Marker is a named empty interface used as an embedded field.
//
// It carries no methods, exercising the utpe.Empty() branch of
// buildNamedEmbedded for interfaces.
type Marker any

// EmbedsEmptyNamedInterface embeds a named empty interface.
//
// swagger:model
type EmbedsEmptyNamedInterface struct {
	Marker

	Value string `json:"value"`
}

// EmbedsError embeds the predeclared error interface, exercising the
// isStdError branch of buildNamedEmbedded.
//
// swagger:model
type EmbedsError struct {
	error

	Code int `json:"code"`
}

// Handler is a non-empty named interface with a single exported method.
type Handler interface {
	// Handle is a unary method exposed as a schema property.
	Handle() string
}

// EmbedsNamedInterface embeds a non-empty, non-error named interface.
//
// swagger:model
type EmbedsNamedInterface struct {
	Handler

	Tag string `json:"tag"`
}
