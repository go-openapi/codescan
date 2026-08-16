// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package sub holds a subtype declared in a package other than the base's, so
// the reverse `swagger:allOf` index is exercised across package boundaries.
package sub

import (
	"github.com/go-openapi/codescan/testdata/enhancements/discriminated-subtypes/base"
)

// ModelY is a cross-package subtype of base.TeslaCar.
//
// swagger:model modelY
type ModelY struct {
	// swagger:allOf
	base.TeslaCar

	// Range in km
	Range int `json:"range"`
}
