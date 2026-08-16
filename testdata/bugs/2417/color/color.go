// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package color

// Color is a base model in the color package.
type Color struct {
	Hue string `json:"hue"`
}

// SamePackageAlias is an alias of Color declared in the same package.
type SamePackageAlias Color
