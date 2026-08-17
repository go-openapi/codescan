// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package metaplacement carries an ordinary package doc that says nothing about the
// specification, so that the meta block below is the only thing the scan may read it
// from.
//
// This sentence used to become the specification's title, because the meta block was
// taken from the file's package doc rather than from the comment the annotation was
// actually written in.
package metaplacement

// The meta block, written below the package clause rather than above it.
//
// Version: 1.2.3
// Host: api.example.com
// BasePath: /v2
// Schemes: https
//
// swagger:meta

// Pet is a pet.
//
// swagger:model Pet
type Pet struct {
	// Name of the pet.
	Name string `json:"name"`
}
