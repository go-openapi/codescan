// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package b

import (
	"github.com/go-openapi/codescan/fixtures/bugs/2417/a"
	"github.com/go-openapi/codescan/fixtures/bugs/2417/color"
)

// Each figure isolates one cell of the issue's embedding × aliasing matrix so
// its contribution is observable (all the source types share the field "hue",
// so a combined struct would collapse them and hide which embed contributed).

// CrossAliasEmbed embeds a cross-package alias anonymously.
// Reporter (old go-swagger): NOT rendered.
//
// swagger:model CrossAliasEmbed
type CrossAliasEmbed struct {
	a.AnotherPackageAlias
}

// PlainEmbed embeds the plain cross-package type anonymously.
// Reporter: rendered.
//
// swagger:model PlainEmbed
type PlainEmbed struct {
	color.Color
}

// SameAliasEmbed embeds a same-package alias anonymously.
// Reporter: rendered.
//
// swagger:model SameAliasEmbed
type SameAliasEmbed struct {
	color.SamePackageAlias
}

// NamedCrossAlias references a cross-package alias as a named field.
// Reporter: rendered.
//
// swagger:model NamedCrossAlias
type NamedCrossAlias struct {
	AnyName a.AnotherPackageAlias `json:"anyName"`
}
