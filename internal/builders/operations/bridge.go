// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package operations

import (
	"fmt"
	"strings"

	"github.com/go-openapi/codescan/internal/parsers"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/loads/fmts"
	oaispec "github.com/go-openapi/spec"
	yaml "go.yaml.in/yaml/v3"
)

// applyBlockToOperation is the grammar-path counterpart of the
// legacy YAMLSpecScanner pipeline used by Builder.Build. It parses
// path.Remaining (the comments following the swagger:operation
// annotation) through the grammar parser and:
//
//  1. splits prose into Summary (first title paragraph) and
//     Description (rest) via v1's CollectScannerTitleDescription on
//     block.ProseLines() — line-preserving `"\n"` join for parity;
//  2. feeds the first YAML block body (captured between the `---`
//     fences by collectYAMLBody) through
//     yaml.Unmarshal → fmts.YAMLToJSON → op.UnmarshalJSON, the
//     same final step as YAMLSpecScanner.UnmarshalSpec.
//
// The grammar's lexer recognises `---` as TokenYAMLFence and the
// lines between fences as TokenRawLine with Raw preserved; the
// grammar's collectYAMLBody joins those raw lines with `\n` into
// RawYAML.Text. Nothing else — title, description, YAML fences —
// needs to round-trip through the legacy scanner.
func (o *Builder) applyBlockToOperation(op *oaispec.Operation) error {
	fset := o.ctx.FileSet()
	block := grammar.NewParser(fset).Parse(o.path.Remaining)

	title, desc := parsers.CollectScannerTitleDescription(block.ProseLines())
	op.Summary = parsers.JoinDropLast(title)
	op.Description = parsers.JoinDropLast(desc)

	var yamlBody string
	for y := range block.YAMLBlocks() {
		yamlBody = y.Text
		break // v1 accepts only one fenced YAML body per operation
	}
	if yamlBody == "" {
		return nil
	}

	return unmarshalOpYAML(yamlBody, op.UnmarshalJSON)
}

// unmarshalOpYAML converts a raw YAML body into the operation's
// JSON-shape expected by oaispec.Operation.UnmarshalJSON. Mirrors
// parsers.YAMLSpecScanner.UnmarshalSpec — common leading indent is
// stripped and tab indentation normalised to spaces so YAML bodies
// written with godoc-style leading tabs (e.g. the go119 fixture)
// parse correctly.
func unmarshalOpYAML(body string, unmarshal func([]byte) error) error {
	lines := strings.Split(body, "\n")
	lines = parsers.RemoveIndent(lines)
	normalized := strings.Join(lines, "\n")

	yamlValue := make(map[any]any)
	if err := yaml.Unmarshal([]byte(normalized), &yamlValue); err != nil {
		return fmt.Errorf("operation yaml body: %w", err)
	}

	jsonValue, err := fmts.YAMLToJSON(yamlValue)
	if err != nil {
		return fmt.Errorf("operation yaml→json: %w", err)
	}

	data, err := jsonValue.MarshalJSON()
	if err != nil {
		return fmt.Errorf("operation json marshal: %w", err)
	}

	return unmarshal(data)
}
