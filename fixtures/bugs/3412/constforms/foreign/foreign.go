// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package foreign declares a type sharing its name with an enum type of the parent package, so the
// membership test cannot be name-only.
package foreign

// Weekday is unrelated to constforms.Weekday beyond the name.
type Weekday int

// TheThirteenth is a member of THIS Weekday, not of the annotated enum next door.
const TheThirteenth Weekday = 13
