// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-openapi/codescan/internal/parsers/yaml"
	oaispec "github.com/go-openapi/spec"
)

const (
	// kvParts is the number of parts when splitting key:value pairs.
	kvParts = 2
)

// Many thanks go to https://github.com/yvasiyarov/swagger
// this is loosely based on that implementation but for swagger 2.0

type matchOnlyParam struct {
	rx *regexp.Regexp
}

func (mo *matchOnlyParam) Matches(line string) bool {
	return mo.rx.MatchString(line)
}

func (mo *matchOnlyParam) Parse(_ []string) error {
	return nil
}

type MatchParamIn struct {
	*matchOnlyParam
}

func NewMatchParamIn(_ *oaispec.Parameter) *MatchParamIn {
	return NewMatchIn()
}

// NewMatchIn returns a match-only tagger that claims `in: <location>`
// lines. The `in:` directive is extracted separately via
// parsers.ParamLocation; this tagger only prevents the line from
// being absorbed into the surrounding description by a SectionedParser.
func NewMatchIn() *MatchParamIn {
	return &MatchParamIn{
		matchOnlyParam: &matchOnlyParam{
			rx: rxIn,
		},
	}
}

type MatchParamRequired struct {
	*matchOnlyParam
}

func NewMatchParamRequired(_ *oaispec.Parameter) *MatchParamRequired {
	return &MatchParamRequired{
		matchOnlyParam: &matchOnlyParam{
			rx: rxRequired,
		},
	}
}

type SetDeprecatedOp struct {
	tgt *oaispec.Operation
}

func NewSetDeprecatedOp(operation *oaispec.Operation) *SetDeprecatedOp {
	return &SetDeprecatedOp{
		tgt: operation,
	}
}

func (su *SetDeprecatedOp) Matches(line string) bool {
	return rxDeprecated.MatchString(line)
}

func (su *SetDeprecatedOp) Parse(lines []string) error {
	if len(lines) == 0 || (len(lines) == 1 && len(lines[0]) == 0) {
		return nil
	}

	matches := rxDeprecated.FindStringSubmatch(lines[0])
	if len(matches) > 1 && len(matches[1]) > 0 {
		req, err := strconv.ParseBool(matches[1])
		if err != nil {
			return err
		}
		su.tgt.Deprecated = req
	}

	return nil
}

type ConsumesDropEmptyParser struct {
	*multilineYAMLListParser
}

func NewConsumesDropEmptyParser(set func([]string)) *ConsumesDropEmptyParser {
	return &ConsumesDropEmptyParser{
		multilineYAMLListParser: &multilineYAMLListParser{
			set: set,
			rx:  rxConsumes,
		},
	}
}

type ProducesDropEmptyParser struct {
	*multilineYAMLListParser
}

func NewProducesDropEmptyParser(set func([]string)) *ProducesDropEmptyParser {
	return &ProducesDropEmptyParser{
		multilineYAMLListParser: &multilineYAMLListParser{
			set: set,
			rx:  rxProduces,
		},
	}
}

type multilineDropEmptyParser struct {
	set func([]string)
	rx  *regexp.Regexp
}

func newMultilineDropEmptyParser(rx *regexp.Regexp, set func([]string)) *multilineDropEmptyParser {
	return &multilineDropEmptyParser{
		set: set,
		rx:  rx,
	}
}

func (m *multilineDropEmptyParser) Matches(line string) bool {
	return m.rx.MatchString(line)
}

func (m *multilineDropEmptyParser) Parse(lines []string) error {
	m.set(removeEmptyLines(lines))

	return nil
}

// multilineYAMLListParser is the Q4 replacement for
// multilineDropEmptyParser on list-valued block bodies
// (`consumes:` / `produces:` in meta + operation scope). The
// body is captured raw — its list-item markers (`- value`)
// survive the preprocessor — and interpreted by
// internal/parsers/yaml/ as a YAML list. Strict list: non-list
// bodies emit a warning and produce no values.
//
// See `.claude/plans/workshops/w2-enum.md` §2.6 (quirk 4),
// `grammar-parser-architecture.md` §3.3 (sub-parser pattern),
// and `.claude/plans/forthcoming-features.md` §5.2 (P7.7 doc
// follow-up).
type multilineYAMLListParser struct {
	set func([]string)
	rx  *regexp.Regexp
}

func (m *multilineYAMLListParser) Matches(line string) bool {
	return m.rx.MatchString(line)
}

func (m *multilineYAMLListParser) Parse(lines []string) error {
	// Strip comment noise but preserve `-` (the YAML list marker).
	// Matches rxUncommentHeaders minus the dash.
	cleaned := cleanupScannerLines(lines, rxUncommentNoDash)

	// Drop drop-empty to avoid blank spacer lines between items
	// confusing the YAML parser.
	cleaned = removeEmptyLines(cleaned)
	if len(cleaned) == 0 {
		return nil
	}

	body := strings.Join(cleaned, "\n")
	parsed, err := yaml.Parse(body)
	if err != nil {
		log.Printf("WARNING: parse.invalid-block-body: %v", err)
		return nil
	}
	list, ok := parsed.([]any)
	if !ok {
		log.Printf("WARNING: parse.invalid-block-body: expected YAML list, got %T", parsed)
		return nil
	}

	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, fmt.Sprintf("%v", item))
	}
	m.set(out)
	return nil
}
