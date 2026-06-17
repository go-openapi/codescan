// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package a

import "github.com/go-openapi/codescan/fixtures/bugs/2417/color"

// AnotherPackageAlias is an alias of color.Color declared in a different
// package than the underlying type.
type AnotherPackageAlias color.Color
