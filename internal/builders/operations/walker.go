// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package operations

import (
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/parsers/yaml"
	oaispec "github.com/go-openapi/spec"
)

// applyBlockToOperation parses path.Remaining through grammar and
// writes Summary / Description / YAML body content onto op.
//
// # Details
//
// See [§walker](./README.md#walker) for the three-step bridge from
// the lexer-classified Block onto op and the single-fenced-body
// contract.
func (o *Builder) applyBlockToOperation(op *oaispec.Operation) error {
	block := grammar.NewParser(o.Ctx.FileSet()).Parse(o.path.Remaining)

	op.Summary = block.Title()
	op.Description = block.Description()

	for y := range block.YAMLBlocks() {
		return yaml.UnmarshalBody(y.Text, op.UnmarshalJSON)
	}
	return nil
}
