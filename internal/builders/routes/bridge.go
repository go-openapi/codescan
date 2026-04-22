// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"github.com/go-openapi/codescan/internal/parsers"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/parsers/helpers"
	oaispec "github.com/go-openapi/spec"
)

// applyBlockToRoute is the grammar-path counterpart of Builder.Build's
// SectionedParser invocation. Parses route.Remaining, extracts
// summary/description, and dispatches each level-0 Property to the
// appropriate setter. Body parsing for the multi-line keywords
// (consumes, produces, security, parameters, responses, extensions)
// delegates to the existing v1 parser instances — each already
// handles its specific body shape (YAML lists, name:value mappings,
// nested extension bodies). The bridge contributes line-splitting /
// title-description / dispatch; the heavy lifting stays in the
// established parser code.
func (r *Builder) applyBlockToRoute(op *oaispec.Operation) error {
	block := grammar.NewParser(r.ctx.FileSet()).Parse(r.route.Remaining)

	title, desc := parsers.CollectScannerTitleDescription(block.ProseLines())
	op.Summary = parsers.JoinDropLast(title)
	op.Description = parsers.JoinDropLast(desc)

	for prop := range block.Properties() {
		if prop.ItemsDepth != 0 {
			continue
		}
		if err := r.dispatchRouteKeyword(prop, op); err != nil {
			return err
		}
	}
	return nil
}

// Keyword names reused from grammar's keyword table — kept as
// constants to avoid magic strings in the dispatch table.
const (
	kwSchemes    = "schemes"
	kwDeprecated = "deprecated"
	kwConsumes   = "consumes"
	kwProduces   = "produces"
	kwSecurity   = "security"
	kwParameters = "parameters"
	kwResponses  = "responses"
	kwExtensions = "extensions"
)

// dispatchRouteKeyword routes one grammar Property to the matching
// body parser. Simple body shapes (schemes comma-list, consumes /
// produces YAML-list, security name:scope lines) use shared
// helpers in internal/parsers/helpers. The three domain-heavy
// body parsers (parameters, responses, extensions) still live in
// internal/parsers/ — their v1-parity logic (e.g. `+ name:` param
// blocks, `200: someResponse` response mapping, nested YAML
// extension maps) is substantial enough to warrant dedicated
// files.
func (r *Builder) dispatchRouteKeyword(p grammar.Property, op *oaispec.Operation) error {
	switch p.Keyword.Name {
	case kwSchemes:
		if v := helpers.SchemesList(p.Value); v != nil {
			op.Schemes = v
		}
	case kwDeprecated:
		if p.Typed.Type == grammar.ValueBoolean {
			op.Deprecated = p.Typed.Boolean
		}
	case kwConsumes:
		op.Consumes = helpers.YAMLListBody(p.Body)
	case kwProduces:
		op.Produces = helpers.YAMLListBody(p.Body)
	case kwSecurity:
		op.Security = helpers.SecurityRequirements(p.Body)
	case kwParameters:
		return parsers.NewSetParams(r.parameters, opParamSetter(op)).Parse(p.Body)
	case kwResponses:
		return parsers.NewSetResponses(r.definitions, r.responses, opResponsesSetter(op)).Parse(p.Body)
	case kwExtensions:
		return parsers.NewSetExtensions(opExtensionsSetter(op), r.ctx.Debug()).Parse(p.Body)
	}
	return nil
}
