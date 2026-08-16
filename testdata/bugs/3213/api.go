// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug3213 reproduces go-swagger issue #3213 ("Consider TypeSpec
// comments"): annotations and doc comments attached to an individual
// TypeSpec inside a grouped `type ( ... )` declaration — rather than to the
// enclosing GenDecl — must still be parsed. Two models share one type group
// with distinct per-spec docs (a GenDecl-only reader could not tell them
// apart), and a grouped enum is referenced by a third model.
package bug3213

type (
	// Alpha is the first grouped model.
	//
	// swagger:model Alpha
	Alpha struct {
		Name string `json:"name"`
	}

	// Beta is the second grouped model, sharing Alpha's type group.
	//
	// swagger:model Beta
	Beta struct {
		// The current status.
		Status Status `json:"status"`
	}
)

type (
	// Status is an enum declared inside a type group.
	//
	// swagger:enum Status
	Status string
)

const (
	StatusOn  Status = "on"
	StatusOff Status = "off"
)
